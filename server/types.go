package server

import (
	"golang.org/x/crypto/ssh"
)

type FileInfoPayload struct {
	FileID    string
	EditUrl   string
	EditToken string
}

func (f *FileInfoPayload) Marshal() []byte {
	return ssh.Marshal(f)
}
