package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"remdit-server/config"

	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	conf *ssh.ServerConfig
}

func (s *SSHServer) HandleConn(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.conf)
	if err != nil {
		slog.Error("failed to create SSH server connection", "err", err)
		return
	}
	slog.Info("SSH connection established",
		"remote_addr", sshConn.RemoteAddr(),
		"user", sshConn.User(),
	)
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported channel type")
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
				case "shell":
					if err := req.Reply(true, nil); err != nil {
						slog.Error("failed to reply to shell request", "err", err)
						return
					}
					// [TODO] main logic here

					return
				default:
					if req.WantReply {
						req.Reply(true, nil)
					}
				}
			}
		}(requests)
	}
}

func Serve(ctx context.Context) error {
	sshConf := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Do nothing
			return nil, nil
		},
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

	addr := fmt.Sprintf(":%d", config.C.SSHPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	slog.Info("SSH server listening on", "addr", addr)

	server := &SSHServer{conf: sshConf}

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
