package filestor

import (
	"context"
	"fmt"
	"os"
	"sync"
)

type FileInfoStorage interface {
	Save(ctx context.Context, fileID string, f File) error
	Get(ctx context.Context, fileID string) File
	Delete(ctx context.Context, fileID string) error
}

type File interface {
	ID() string
	Path() string
	Name() string
	Close() error
}

type fileImpl struct {
	id   string
	path string
	name string
}

func (f *fileImpl) ID() string {
	return f.id
}
func (f *fileImpl) Path() string {
	return f.path
}
func (f *fileImpl) Name() string {
	return f.name
}

func (f *fileImpl) Close() error {
	return os.RemoveAll(f.path)
}

func NewFileInfo(id, path, name string) File {
	return &fileImpl{
		id:   id,
		path: path,
		name: name,
	}
}

type FileMemoryStorage struct {
	data map[string]File
	mu   sync.RWMutex
}

var _ FileInfoStorage = (*FileMemoryStorage)(nil)

var defaultStor FileInfoStorage = NewFileMemoryStorage()

func Default() FileInfoStorage {
	if defaultStor == nil {
		defaultStor = NewFileMemoryStorage()
	}
	return defaultStor
}

func NewFileMemoryStorage() *FileMemoryStorage {
	return &FileMemoryStorage{
		data: make(map[string]File),
	}
}

func (s *FileMemoryStorage) Save(ctx context.Context, fileID string, f File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[fileID] = f
	return nil
}

func (s *FileMemoryStorage) Get(ctx context.Context, fileID string) File {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, exists := s.data[fileID]
	if !exists {
		return nil
	}
	return info
}

func (s *FileMemoryStorage) Delete(ctx context.Context, fileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if file, exists := s.data[fileID]; exists {
		file.Close()
		delete(s.data, fileID)
	}
	return nil
}

func Save(ctx context.Context, fileID string, f File) error {
	if f == nil {
		return fmt.Errorf("file cannot be nil")
	}
	return defaultStor.Save(ctx, fileID, f)
}

func Get(ctx context.Context, fileID string) File {
	if fileID == "" {
		return nil
	}
	return defaultStor.Get(ctx, fileID)
}

func Delete(ctx context.Context, fileID string) error {
	if fileID == "" {
		return fmt.Errorf("file ID cannot be empty")
	}
	return defaultStor.Delete(ctx, fileID)
}
