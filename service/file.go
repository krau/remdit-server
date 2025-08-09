package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/retry"
)

type FileInfoStorage interface {
	Save(ctx context.Context, fileID string, info FileInfo) error
	Get(ctx context.Context, fileID string) FileInfo
	Delete(ctx context.Context, fileID string) error
}

type FileInfo interface {
	ID() string
	Path() string
	Name() string
}

type FileInfoMemoryStorage struct {
	data map[string]FileInfo
	mu   sync.RWMutex
}

func NewFileInfoMemoryStorage() *FileInfoMemoryStorage {
	return &FileInfoMemoryStorage{
		data: make(map[string]FileInfo),
	}
}

func (s *FileInfoMemoryStorage) Save(ctx context.Context, fileID string, info FileInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[fileID] = info
	return nil
}

func (s *FileInfoMemoryStorage) Get(ctx context.Context, fileID string) FileInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, exists := s.data[fileID]
	if !exists {
		return nil
	}
	return info
}

func (s *FileInfoMemoryStorage) Delete(ctx context.Context, fileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, fileID)
	return nil
}

// 将文件写入到本地并同步给客户端
func WriteAndSyncFile(ctx context.Context, stor FileInfoStorage, fileID string, content []byte) error {
	fileInfo := stor.Get(ctx, fileID)
	if fileInfo == nil {
		return fmt.Errorf("file not found")
	}
	conn, ok := GetSSHConn(fileID)
	if !ok {
		return fmt.Errorf("no active SSH connection for this file")
	}
	if err := os.WriteFile(fileInfo.Path(), content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return retry.Retry(func() error {
		ok, _, err := conn.SendRequest("file-save", true, content)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("file-save request was rejected")
		}
		return nil
	}, retry.Context(ctx),
		retry.RetryWithLinearBackoff(time.Microsecond*50))
}
