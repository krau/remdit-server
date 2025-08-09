package service

import (
	"context"
	"sync"
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
