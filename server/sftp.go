package server

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
)

// TempFileHandler 用于处理临时文件操作
type TempFileHandler struct {
	tempDir string
}

func (h *TempFileHandler) TempDir() string {
	return h.tempDir
}

func (h *TempFileHandler) Clean() error {
	if err := os.RemoveAll(h.tempDir); err != nil {
		slog.Error("failed to clean temp directory", "dir", h.tempDir, "err", err)
		return err
	}
	slog.Info("cleaned temp directory", "dir", h.tempDir)
	return nil
}

// NewTempFileHandler 创建新的临时文件处理器
func NewTempFileHandler() *TempFileHandler {
	tempDir := filepath.Join("remdit-uploads", uuid.New().String())
	os.MkdirAll(tempDir, 0755)
	return &TempFileHandler{
		tempDir: tempDir,
	}
}

// Fileread 实现文件读取
func (h *TempFileHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	slog.Info("SFTP read request", "path", r.Filepath)
	fullPath := filepath.Join(h.tempDir, r.Filepath)
	return os.Open(fullPath)
}

// Filewrite 实现文件写入，将客户端文件写入临时文件
func (h *TempFileHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	slog.Info("SFTP write request", "path", r.Filepath)

	// 确保目录存在
	fullPath := filepath.Join(h.tempDir, r.Filepath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create directory", "dir", dir, "err", err)
		return nil, err
	}

	// 创建临时文件
	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error("failed to create temp file", "path", fullPath, "err", err)
		return nil, err
	}

	slog.Info("created temp file for client upload", "temp_path", fullPath)
	return file, nil
}

// Filecmd 实现文件命令操作
func (h *TempFileHandler) Filecmd(r *sftp.Request) error {
	slog.Info("SFTP command request", "method", r.Method, "path", r.Filepath)

	fullPath := filepath.Join(h.tempDir, r.Filepath)

	switch r.Method {
	case "Remove":
		return os.Remove(fullPath)
	case "Mkdir":
		return os.MkdirAll(fullPath, 0755)
	case "Rename":
		oldPath := filepath.Join(h.tempDir, r.Filepath)
		newPath := filepath.Join(h.tempDir, r.Target)
		return os.Rename(oldPath, newPath)
	case "Rmdir":
		return os.Remove(fullPath)
	case "Setstat":
		// 设置文件属性，这里简单处理
		if r.AttrFlags().Size {
			return os.Truncate(fullPath, int64(r.Attributes().Size))
		}
		return nil
	default:
		slog.Warn("unsupported SFTP command", "method", r.Method)
		return fmt.Errorf("unsupported command: %s", r.Method)
	}
}

// Filelist 实现文件列表操作
func (h *TempFileHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	slog.Info("SFTP list request", "path", r.Filepath)

	fullPath := filepath.Join(h.tempDir, r.Filepath)

	switch r.Method {
	case "List":
		// 列出目录内容
		files, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, err
		}

		var fileInfos []os.FileInfo
		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				continue
			}
			fileInfos = append(fileInfos, info)
		}

		return listerat(fileInfos), nil

	case "Stat":
		// 获取文件状态
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, err
		}
		return listerat([]os.FileInfo{info}), nil

	default:
		return nil, fmt.Errorf("unsupported list method: %s", r.Method)
	}
}

// listerat 实现 ListerAt 接口
type listerat []os.FileInfo

func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}
