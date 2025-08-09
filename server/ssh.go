package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"remdit-server/config"
	"remdit-server/service"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	conf         *ssh.ServerConfig
	fileInfoStor service.FileInfoStorage
}

type SessionState uint

const (
	SessionStateNone SessionState = iota
	SessionStateFileUpload
	SessionStateFileInfo
	SessionStateListen
)

type ConnHandler struct {
	conn         net.Conn
	conf         *ssh.ServerConfig
	fileInfoStor service.FileInfoStorage
	state        SessionState
	file         *TempFileHandler
	serverConn   *ssh.ServerConn
}

func NewConnHandler(conn net.Conn, conf *ssh.ServerConfig, stor service.FileInfoStorage) *ConnHandler {
	return &ConnHandler{
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

	return nil
}

func (h *ConnHandler) Handle(ctx context.Context) error {
	conn := h.conn

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, h.conf)
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
				switch req.Type {
				case "subsystem":
					if h.state != SessionStateNone {
						req.Reply(false, nil)
						continue
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

					req.Reply(true, nil)

					sftpHandlers := sftp.Handlers{FileGet: handler, FilePut: handler, FileCmd: handler, FileList: handler}
					requestServer := sftp.NewRequestServer(channel, sftpHandlers)
					defer requestServer.Close()
					slog.Info("starting SFTP server for client", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())

					if err := requestServer.Serve(); err != nil {
						if err != io.EOF {
							slog.Error("SFTP server error", "err", err)
						}
					}
					slog.Info("SFTP server session ended", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())
					h.state = SessionStateFileUpload

					return
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}(requests)
	}
	slog.Info("SSH session ended", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())
	return nil
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

func (s *SSHServer) HandleConn(ctx context.Context, conn net.Conn) {
	handler := NewConnHandler(conn, s.conf, s.fileInfoStor)
	defer handler.Close()
	if err := handler.Handle(ctx); err != nil {
		slog.Error("failed to handle SSH connection", "err", err)
	}
	slog.Info("SSH connection closed", "remote_addr", conn.RemoteAddr())
}

func Serve(ctx context.Context, stor service.FileInfoStorage) error {
	sshConf := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
		PublicKeyAuthAlgorithms: []string{ssh.KeyAlgoED25519},
	}
	priKey, err := os.ReadFile(config.C.SSHPrivateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(priKey)
	if err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}
	sshConf.AddHostKey(signer)

	addr := fmt.Sprintf("%s:%d", config.C.SSHHost, config.C.SSHPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer ln.Close()
	slog.Info("SSH server listening on", "addr", addr)

	server := &SSHServer{conf: sshConf, fileInfoStor: stor}

	go func() {
		for {
			rawConn, err := ln.Accept()
			if err != nil {
				slog.Error("accept error", "err", err)
				return
			}
			go server.HandleConn(ctx, rawConn)
		}
	}()

	<-ctx.Done()
	slog.Info("SSH server is shutting down")
	return nil
}
