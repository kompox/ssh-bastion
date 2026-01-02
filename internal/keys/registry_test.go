package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestComputeFingerprint(t *testing.T) {
	testKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testKey))
	if err != nil {
		t.Fatalf("Failed to parse test key: %v", err)
	}

	fingerprint := computeFingerprint(pubKey)
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Errorf("Expected fingerprint to start with SHA256:, got %s", fingerprint)
	}
}

func TestAddKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	testKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"

	key, err := registry.AddKey("test-user-id", testKey)
	if err != nil {
		t.Fatalf("Failed to add key: %v", err)
	}

	if key.Fingerprint == "" {
		t.Error("Expected fingerprint to be set")
	}

	if !strings.HasPrefix(key.Fingerprint, "SHA256:") {
		t.Errorf("Expected fingerprint to start with SHA256:, got %s", key.Fingerprint)
	}

	if !key.Enabled {
		t.Error("Expected key to be enabled by default")
	}

	filename := strings.ReplaceAll(strings.ReplaceAll(key.Fingerprint, ":", "_"), "/", "-")
	jsonPath := filepath.Join(tmpDir, "users", "test-user-id", "keys", filename+".json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("Expected JSON metadata file to exist")
	}

	pubPath := filepath.Join(tmpDir, "users", "test-user-id", "keys", filename+".pub")
	if _, err := os.Stat(pubPath); os.IsNotExist(err) {
		t.Error("Expected pub file to exist")
	}
}

func TestInvalidKeyRejected(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	invalidKey := "not-a-valid-key"

	_, err := registry.AddKey("test-user-id", invalidKey)
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}

func TestListKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	testKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"

	_, err := registry.AddKey("test-user-id", testKey)
	if err != nil {
		t.Fatalf("Failed to add key: %v", err)
	}

	keys, err := registry.ListKeys("test-user-id")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}
}

func TestEnableDisableKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	testKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"

	key, err := registry.AddKey("test-user-id", testKey)
	if err != nil {
		t.Fatalf("Failed to add key: %v", err)
	}

	if err := registry.UpdateKeyStatus("test-user-id", key.Fingerprint, false); err != nil {
		t.Fatalf("Failed to disable key: %v", err)
	}

	keys, err := registry.ListKeys("test-user-id")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if keys[0].Enabled {
		t.Error("Expected key to be disabled")
	}

	if err := registry.UpdateKeyStatus("test-user-id", key.Fingerprint, true); err != nil {
		t.Fatalf("Failed to enable key: %v", err)
	}

	keys, err = registry.ListKeys("test-user-id")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if !keys[0].Enabled {
		t.Error("Expected key to be enabled")
	}
}
