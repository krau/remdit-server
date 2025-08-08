package server

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"remdit-server/config"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	maxUploadSize = 10 * 1024 * 1024 // 10 MB
)

type TempFileHandler struct {
	serverConn *ssh.ServerConn
	randomID   string
	tempDir    string
	fileName   string
	uploaded   bool
}

func (h *TempFileHandler) WriteFile(b []byte) error {
	if !h.uploaded {
		return errors.New("file not uploaded yet")
	}
	fullPath := filepath.Join(h.tempDir, h.fileName)
	if err := os.WriteFile(fullPath, b, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	ok, _, err := h.serverConn.SendRequest("file-save", true, b)
	if err != nil {
		return fmt.Errorf("failed to send file-save request: %w", err)
	}
	if !ok {
		return errors.New("file-save request was rejected")
	}
	return nil
}

func (h *TempFileHandler) ID() string {
	return h.randomID
}

func (h *TempFileHandler) Path() string {
	return filepath.Join(h.tempDir, h.fileName)
}

func (h *TempFileHandler) Name() string {
	return h.fileName
}

// NewTempFileHandler creates a handler with a unique temp directory
func NewTempFileHandler(id string, serverConn *ssh.ServerConn) *TempFileHandler {
	tempDir := filepath.Join(config.C.UploadsDir, id)
	return &TempFileHandler{
		randomID:   id,
		tempDir:    tempDir,
		serverConn: serverConn,
	}
}

// Close cleans up the temp directory
func (h *TempFileHandler) Close() error {
	if err := os.RemoveAll(h.tempDir); err != nil {
		slog.Error("failed to clean temp directory", "dir", h.tempDir, "err", err)
		return err
	}
	slog.Debug("cleaned temp directory", "dir", h.tempDir)
	return nil
}

// Fileread only succeeds if the file has been uploaded
func (h *TempFileHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	if !h.uploaded {
		return nil, errors.New("file not uploaded yet")
	}
	slog.Debug("SFTP read request", "path", r.Filepath)
	fullPath := filepath.Join(h.tempDir, h.fileName)
	f, err := os.Open(fullPath)
	if err != nil {
		slog.Error("failed to open file", "path", fullPath, "err", err)
		return nil, errors.New("file not found")
	}
	return f, nil
}

// Filewrite allows a single file upload, with size and name checks
func (h *TempFileHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	// Reject if already uploaded
	if h.uploaded {
		return nil, errors.New("file already uploaded")
	}

	slog.Debug("SFTP write request", "path", r.Filepath)
	// Assign filename and create directories
	if err := os.MkdirAll(h.tempDir, 0755); err != nil {
		slog.Error("failed to create temp directory", "dir", h.tempDir, "err", err)
	}
	h.fileName = filepath.Base(r.Filepath)
	fullPath := filepath.Join(h.tempDir, h.fileName)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create directory", "dir", dir, "err", err)
		return nil, err
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error("failed to create temp file", "path", fullPath, "err", err)
		return nil, err
	}

	slog.Info("created temp file for client upload", "temp_path", fullPath)

	// Wrap writer to enforce max size per write
	writer := &limitedWriterAt{File: file}

	// Mark as uploaded to prevent another Filewrite
	h.uploaded = true
	return writer, nil
}

// Filecmd rejects unsupported commands
func (h *TempFileHandler) Filecmd(r *sftp.Request) error {
	return fmt.Errorf("unsupported command: %s", r.Method)
}

// Filelist rejects directory listing
func (h *TempFileHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return nil, fmt.Errorf("unsupported list method: %s", r.Method)
}

// limitedWriterAt enforces max upload size per write and cumulative
// based on file offset and length

type limitedWriterAt struct {
	*os.File
}

func (w *limitedWriterAt) WriteAt(p []byte, off int64) (int, error) {
	end := off + int64(len(p))
	if end > maxUploadSize {
		slog.Warn("upload size exceeds limit", "offset", off, "length", len(p))
		return 0, fmt.Errorf("upload exceeds max size of %d bytes", maxUploadSize)
	}
	return w.File.WriteAt(p, off)
}
