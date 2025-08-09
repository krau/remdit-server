package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"remdit-server/config"
	"remdit-server/service"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
)

type ConnHandler struct {
	ctx          context.Context
	conn         net.Conn
	conf         *ssh.ServerConfig
	fileInfoStor service.FileInfoStorage
	file         *TempFileHandler
	serverConn   *ssh.ServerConn
	state        SessionState
}

func NewConnHandler(ctx context.Context, conn net.Conn, conf *ssh.ServerConfig, stor service.FileInfoStorage) *ConnHandler {
	return &ConnHandler{
		ctx:          ctx,
		conn:         conn,
		conf:         conf,
		fileInfoStor: stor,
	}
}

func (h *ConnHandler) Close() error {
	errs := make([]error, 0)
	if h.file != nil {
		if err := h.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close temp file handler: %w", err))
		}
	}
	if h.file != nil && h.fileInfoStor != nil {
		if err := h.fileInfoStor.Delete(context.Background(), h.file.ID()); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete file info: %w", err))
		}
	}
	if h.serverConn != nil {
		if err := h.serverConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close SSH server connection: %w", err))
		}
	}
	if h.conn != nil {
		if err := h.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}

	return multierr.Combine(errs...)
}

func (h *ConnHandler) Handle(ctx context.Context) error {
	sshConn, chans, reqs, err := ssh.NewServerConn(h.conn, h.conf)
	if err != nil {
		slog.Error("failed to create SSH server connection", "err", err)
		return err
	}
	h.serverConn = sshConn
	slog.Info(
		"SSH connection established",
		"remote_addr", sshConn.RemoteAddr(),
		"user", sshConn.User(),
	)
	h.state = SessionStateNone

	fileID := uuid.New()

	handler := NewTempFileHandler(fileID.String())
	h.file = handler

	go h.HandleGlobalReqs(ctx, reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		if h.state != SessionStateNone {
			newChan.Reject(ssh.Prohibited, "session already in progress")
			continue
		}
		channel, requests, err := newChan.Accept()
		if err != nil {
			slog.Error("failed to accept channel", "err", err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer channel.Close()
			for req := range in {
				h.HandleChannelReq(req, channel)
			}
		}(requests)
	}
	slog.Info("SSH session ended", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())
	return nil
}

func (h *ConnHandler) HandleChannelReq(req *ssh.Request, channel ssh.Channel) {
	switch req.Type {
	case "subsystem":
		if h.state != SessionStateNone {
			req.Reply(false, nil)
			return
		}
		var payload struct {
			Name string
		}
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			slog.Error("failed to unmarshal subsystem request", "err", err)
			req.Reply(false, nil)
			return
		}
		if payload.Name != "sftp" {
			req.Reply(false, nil)
			return
		}

		if err := req.Reply(true, nil); err != nil {
			slog.Error("failed to reply to subsystem request", "err", err)
			return
		}

		sftpHandlers := sftp.Handlers{FileGet: h.file, FilePut: h.file, FileCmd: h.file, FileList: h.file}
		requestServer := sftp.NewRequestServer(channel, sftpHandlers)
		defer requestServer.Close()
		slog.Info("starting SFTP server for client", "remote_addr", h.serverConn.RemoteAddr(), "user", h.serverConn.User())

		if err := requestServer.Serve(); err != nil {
			if err != io.EOF {
				slog.Error("SFTP server error", "err", err)
			}
		}
		slog.Info("SFTP server session ended", "remote_addr", h.serverConn.RemoteAddr(), "user", h.serverConn.User())

		h.state = SessionStateFileUpload
		req.Reply(true, nil)
		return
	default:
		if req.WantReply {
			req.Reply(false, nil)
		}
	}
}

func (h *ConnHandler) HandleGlobalReqs(ctx context.Context, reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case "file-info":
			if h.state != SessionStateFileUpload {
				slog.Warn("file-info request received in wrong state", "state", h.state)
				req.Reply(false, nil)
				continue
			}

			payload := &FileInfoMessagePayload{
				FileID:  h.file.ID(),
				EditUrl: fmt.Sprintf("%s/edit/%s", config.C.ServerURLs[rand.Intn(len(config.C.ServerURLs))], h.file.ID()),
			}

			req.Reply(true, ssh.Marshal(payload))
			h.state = SessionStateFileInfo
		case "listen":
			if h.state != SessionStateFileInfo {
				slog.Warn("listen request received in wrong state", "state", h.state)
				req.Reply(false, nil)
				continue
			}
			if err := h.fileInfoStor.Save(context.Background(), h.file.ID(), h.file); err != nil {
				slog.Error("failed to save file info", "err", err)
				req.Reply(false, nil)
				return
			}
			service.AddSSHConn(h.file.ID(), h.serverConn)
			defer service.RemoveSSHConn(h.file.ID())
			req.Reply(true, nil)
			h.state = SessionStateListen
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}
