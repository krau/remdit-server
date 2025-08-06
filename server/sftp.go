package server

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
)

type TempFileHandler struct {
	randomID string
	tempDir  string
	fileName string
	uploaded bool
}

func (h *TempFileHandler) Close() error {
	if err := os.RemoveAll(h.tempDir); err != nil {
		slog.Error("failed to clean temp directory", "dir", h.tempDir, "err", err)
		return err
	}
	slog.Debug("cleaned temp directory", "dir", h.tempDir)
	return nil
}

func NewTempFileHandler() *TempFileHandler {
	randomID := uuid.New().String()
	tempDir := filepath.Join("remdit-uploads", randomID)
	os.MkdirAll(tempDir, 0755)
	return &TempFileHandler{
		randomID: randomID,
		tempDir:  tempDir,
	}
}

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

func (h *TempFileHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	if h.uploaded {
		return nil, errors.New("file already uploaded")
	}
	slog.Debug("SFTP write request", "path", r.Filepath)
	fileName := filepath.Base(r.Filepath)
	h.fileName = fileName
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
	return file, nil
}

func (h *TempFileHandler) Filecmd(r *sftp.Request) error {
	return fmt.Errorf("unsupported command: %s", r.Method)
}

func (h *TempFileHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return nil, fmt.Errorf("unsupported list method: %s", r.Method)
}
