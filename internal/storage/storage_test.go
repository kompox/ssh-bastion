package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := New(tmpDir)

	testData := []byte("test data")
	relPath := "test/file.txt"

	if err := store.AtomicWrite(relPath, testData); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	fullPath := store.Path(relPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("Expected file to exist after atomic write")
	}

	readData, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Expected %q, got %q", testData, readData)
	}
}

func TestAtomicWriteCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	store := New(tmpDir)

	testData := []byte("test data")
	relPath := "deep/nested/directory/file.txt"

	if err := store.AtomicWrite(relPath, testData); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	dirPath := filepath.Join(tmpDir, "deep", "nested", "directory")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}
}

func TestAtomicWriteOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := New(tmpDir)

	relPath := "test/file.txt"

	if err := store.AtomicWrite(relPath, []byte("first")); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	if err := store.AtomicWrite(relPath, []byte("second")); err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	readData, err := os.ReadFile(store.Path(relPath))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readData) != "second" {
		t.Errorf("Expected 'second', got %q", readData)
	}
}
