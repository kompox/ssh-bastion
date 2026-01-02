package keys

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/kompox/ssh-bastion/internal/storage"
)

type Key struct {
	Fingerprint string    `json:"fingerprint"`
	PublicKey   string    `json:"publicKey"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Registry struct {
	store *storage.Store
}

func NewRegistry(store *storage.Store) *Registry {
	return &Registry{store: store}
}

func (r *Registry) AddKey(userDirID, publicKeyStr string) (*Key, error) {
	publicKeyStr = strings.TrimSpace(publicKeyStr)

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyStr))
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	fingerprint := computeFingerprint(pubKey)

	key := &Key{
		Fingerprint: fingerprint,
		PublicKey:   publicKeyStr,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := r.saveKey(userDirID, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (r *Registry) ListKeys(userDirID string) ([]*Key, error) {
	keysDir := r.store.Path(filepath.Join("users", userDirID, "keys"))

	entries, err := os.ReadDir(keysDir)
	if os.IsNotExist(err) {
		return []*Key{}, nil
	}
	if err != nil {
		return nil, err
	}

	var keys []*Key
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(keysDir, entry.Name()))
		if err != nil {
			continue
		}

		var key Key
		if err := json.Unmarshal(data, &key); err != nil {
			continue
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

func (r *Registry) UpdateKeyStatus(userDirID, fingerprint string, enabled bool) error {
	key, err := r.getKey(userDirID, fingerprint)
	if err != nil {
		return err
	}

	key.Enabled = enabled
	key.UpdatedAt = time.Now().UTC()

	return r.saveKey(userDirID, key)
}

func (r *Registry) DeleteKey(userDirID, fingerprint string) error {
	// Match UpdateKeyStatus behavior: deleting a non-existent key should be an error
	// so the caller can present a meaningful message.
	if _, err := r.getKey(userDirID, fingerprint); err != nil {
		return err
	}

	filename := fingerprintToFilename(fingerprint)
	basePath := filepath.Join("users", userDirID, "keys", filename)

	if err := os.Remove(r.store.Path(basePath + ".json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(r.store.Path(basePath + ".pub")); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (r *Registry) getKey(userDirID, fingerprint string) (*Key, error) {
	filename := fingerprintToFilename(fingerprint)
	path := filepath.Join("users", userDirID, "keys", filename+".json")
	data, err := os.ReadFile(r.store.Path(path))
	if err != nil {
		return nil, err
	}

	var key Key
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (r *Registry) saveKey(userDirID string, key *Key) error {
	filename := fingerprintToFilename(key.Fingerprint)
	basePath := filepath.Join("users", userDirID, "keys", filename)

	metadata, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return err
	}

	if err := r.store.AtomicWrite(basePath+".json", metadata); err != nil {
		return err
	}

	if err := r.store.AtomicWrite(basePath+".pub", []byte(key.PublicKey)); err != nil {
		return err
	}

	return nil
}

func computeFingerprint(pubKey ssh.PublicKey) string {
	hash := sha256.Sum256(pubKey.Marshal())
	encoded := base64.RawStdEncoding.EncodeToString(hash[:])
	return "SHA256:" + encoded
}

func fingerprintToFilename(fingerprint string) string {
	return strings.ReplaceAll(strings.ReplaceAll(fingerprint, ":", "_"), "/", "-")
}

func (r *Registry) RenderAuthorizedKeys() error {
	var allKeys []string

	usersDir := r.store.Path("users")
	userEntries, err := os.ReadDir(usersDir)
	if os.IsNotExist(err) {
		userEntries = nil
	} else if err != nil {
		return err
	}

	for _, userEntry := range userEntries {
		if !userEntry.IsDir() {
			continue
		}

		userDirID := userEntry.Name()
		keys, err := r.ListKeys(userDirID)
		if err != nil {
			continue
		}

		for _, key := range keys {
			if key.Enabled {
				allKeys = append(allKeys, key.PublicKey)
			}
		}
	}

	content := strings.Join(allKeys, "\n")
	if content != "" {
		content += "\n"
	}

	return r.store.AtomicWrite(filepath.Join("authorized_keys", "jump"), []byte(content))
}
