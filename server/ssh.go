package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"os"
	"remdit-server/config"
	"remdit-server/service"

	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	conf         *ssh.ServerConfig
	fileInfoStor service.FileInfoStorage
}

func (s *SSHServer) HandleConn(ctx context.Context, conn net.Conn) {
	handler := NewConnHandler(ctx, conn, s.conf, s.fileInfoStor)
	defer handler.Close()
	if err := handler.Handle(ctx); err != nil {
		slog.Error("failed to handle SSH connection", "err", err)
	}
	slog.Info("SSH connection closed", "remote_addr", conn.RemoteAddr())
}

func NewSSHServerConfig() *ssh.ServerConfig {
	if config.C.SSHPasswordAuth {
		return &ssh.ServerConfig{
			PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
				slog.Info("SSH password authentication attempt",
					"user", conn.User(),
					"remote_addr", conn.RemoteAddr())

				for _, allowedPassword := range config.C.SSHAllowedPasswords {
					if allowedPassword == "" {
						continue
					}
					if subtle.ConstantTimeCompare([]byte(password), []byte(allowedPassword)) == 1 {
						slog.Info("SSH authentication successful",
							"user", conn.User(),
							"remote_addr", conn.RemoteAddr())
						return &ssh.Permissions{}, nil
					}
				}
				slog.Warn("SSH authentication failed: invalid password",
					"user", conn.User(),
					"remote_addr", conn.RemoteAddr())
				return nil, fmt.Errorf("invalid password")
			},
			AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
				if err != nil {
					slog.Warn("SSH authentication failed",
						"user", conn.User(),
						"method", method,
						"remote_addr", conn.RemoteAddr(),
						"err", err)
				}
			},
		}
	}
	return &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
		PublicKeyAuthAlgorithms: []string{ssh.KeyAlgoED25519},
	}
}

func Serve(ctx context.Context, stor service.FileInfoStorage) error {
	sshConf := NewSSHServerConfig()
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
