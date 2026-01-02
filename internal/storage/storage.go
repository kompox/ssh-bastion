package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	DataDir string
}

func New(dataDir string) *Store {
	return &Store{DataDir: dataDir}
}

func (s *Store) EnsureDir(relPath string) error {
	fullPath := filepath.Join(s.DataDir, relPath)
	return os.MkdirAll(fullPath, 0755)
}

func (s *Store) Path(relPath string) string {
	return filepath.Join(s.DataDir, relPath)
}

func (s *Store) AtomicWrite(relPath string, data []byte) error {
	fullPath := filepath.Join(s.DataDir, relPath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("sync: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}
