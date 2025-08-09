package server

import (
	"context"
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
