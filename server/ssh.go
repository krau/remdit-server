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

func (s *SSHServer) HandleConn(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.conf)
	if err != nil {
		slog.Error("failed to create SSH server connection", "err", err)
		return
	}
	defer sshConn.Close()
	slog.Info(
		"SSH connection established",
		"remote_addr", sshConn.RemoteAddr(),
		"user", sshConn.User(),
	)
	var sessionState SessionState = SessionStateNone
	fileID := uuid.New()

	defer func() {
		if err := s.fileInfoStor.Delete(context.Background(), fileID.String()); err != nil {
			slog.Error("failed to delete file info", "err", err)
		}
	}()

	handler := NewTempFileHandler(fileID.String())
	defer handler.Close()

	go func(in <-chan *ssh.Request) {
		for req := range in {
			switch req.Type {
			case "file-info":
				if sessionState != SessionStateFileUpload {
					slog.Warn("file-info request received in wrong state", "state", sessionState)
					req.Reply(false, nil)
					continue
				}

				fileInfo := s.fileInfoStor.Get(context.Background(), fileID.String())
				if fileInfo == nil {
					slog.Error("file info not found", "fileID", fileID.String())
					req.Reply(false, nil)
					continue
				}

				payload := &FileInfoMessagePayload{
					FileID:  fileInfo.ID(),
					EditUrl: fmt.Sprintf("%s/edit/%s", config.C.ServerURLs[rand.Intn(len(config.C.ServerURLs))], fileID.String()),
				}

				req.Reply(true, ssh.Marshal(payload))
				sessionState = SessionStateFileInfo
			case "listen":
				if sessionState != SessionStateFileInfo {
					slog.Warn("listen request received in wrong state", "state", sessionState)
					req.Reply(false, nil)
					continue
				}
				if err := s.fileInfoStor.Save(context.Background(), fileID.String(), handler); err != nil {
					slog.Error("failed to save file info", "err", err)
					req.Reply(false, nil)
					return
				}
				req.Reply(true, nil)
				sessionState = SessionStateListen

				// [TODO] 监听 api 的事件并发送到客户端

			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		if sessionState != SessionStateNone {
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
					if sessionState != SessionStateNone {
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

					slog.Info("starting SFTP server for client", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())

					if err := requestServer.Serve(); err != nil {
						if err != io.EOF {
							slog.Error("SFTP server error", "err", err)
						}
					}
					slog.Info("SFTP server session ended", "remote_addr", sshConn.RemoteAddr(), "user", sshConn.User())
					sessionState = SessionStateFileUpload

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
	slog.Info("SSH server listening on", "addr", addr)

	server := &SSHServer{conf: sshConf, fileInfoStor: stor}

	go func() {
		for {
			rawConn, err := ln.Accept()
			if err != nil {
				slog.Error("accept error", "err", err)
				return
			}
			go server.HandleConn(rawConn)
		}
	}()

	<-ctx.Done()
	slog.Info("SSH server is shutting down")
	ln.Close()
	return nil
}
