package sshconn

import (
	"sync"

	"golang.org/x/crypto/ssh"
)

type SSHConnStore struct {
	conns map[string]*ssh.ServerConn
	mu    sync.RWMutex
}

func NewSSHConnStore() *SSHConnStore {
	return &SSHConnStore{
		conns: make(map[string]*ssh.ServerConn),
	}
}

var (
	sshConnStore = NewSSHConnStore()
)

func (s *SSHConnStore) Add(id string, conn *ssh.ServerConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[id] = conn
}

func (s *SSHConnStore) Get(id string) (*ssh.ServerConn, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, exists := s.conns[id]
	return conn, exists
}

func (s *SSHConnStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, id)
}

func Add(id string, conn *ssh.ServerConn) {
	sshConnStore.Add(id, conn)
}

func Get(id string) (*ssh.ServerConn, bool) {
	return sshConnStore.Get(id)
}

func Remove(id string) {
	sshConnStore.Remove(id)
}
