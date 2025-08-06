package server

import (
	"context"
	"sync"
)

type FileInfoStorage interface {
	Save(ctx context.Context, fileID string, info FileInfoPayload) error
	Get(ctx context.Context, fileID string) *FileInfoPayload
	Delete(ctx context.Context, fileID string) error
}

type FileInfoMemoryStorage struct {
	data map[string]FileInfoPayload
	mu   sync.RWMutex
}

func NewFileInfoMemoryStorage() *FileInfoMemoryStorage {
	return &FileInfoMemoryStorage{
		data: make(map[string]FileInfoPayload),
	}
}

func (s *FileInfoMemoryStorage) Save(ctx context.Context, fileID string, info FileInfoPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[fileID] = info
	return nil
}

func (s *FileInfoMemoryStorage) Get(ctx context.Context, fileID string) *FileInfoPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, exists := s.data[fileID]
	if !exists {
		return nil
	}
	return &info
}

func (s *FileInfoMemoryStorage) Delete(ctx context.Context, fileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, fileID)
	return nil
}
