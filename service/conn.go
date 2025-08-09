package service

import (
	"sync"

	"golang.org/x/crypto/ssh"
)

type SSHConnPool struct {
	conns map[string]*ssh.ServerConn
	mu    sync.RWMutex
}

func NewSSHConnPool() *SSHConnPool {
	return &SSHConnPool{
		conns: make(map[string]*ssh.ServerConn),
	}
}

var (
	sshConnPool = NewSSHConnPool()
)

func (p *SSHConnPool) Add(id string, conn *ssh.ServerConn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[id] = conn
}

func (p *SSHConnPool) Get(id string) (*ssh.ServerConn, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	conn, exists := p.conns[id]
	return conn, exists
}

func (p *SSHConnPool) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.conns, id)
}

func AddSSHConn(id string, conn *ssh.ServerConn) {
	sshConnPool.Add(id, conn)
}

func GetSSHConn(id string) (*ssh.ServerConn, bool) {
	return sshConnPool.Get(id)
}

func RemoveSSHConn(id string) {
	sshConnPool.Remove(id)
}
