package web

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kompox/ssh-bastion/internal/keys"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	dir := wd
	for {
		candidate := filepath.Join(dir, "web", "templates", "keys.html")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

func TestKeysTemplate_URLencodesFingerprintInActions(t *testing.T) {
	root := findRepoRoot(t)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	fingerprint := "SHA256:abc/def+ghi=="
	data := map[string]any{
		"Title": "SSH Keys",
		"Email": "test@example.com",
		"Page":  "keys",
		"Keys": []*keys.Key{{
			Fingerprint: fingerprint,
			Enabled:     true,
			CreatedAt:   time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
		}},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	html := buf.String()
	if !bytes.Contains([]byte(html), []byte("/ssh/keys/SHA256%3Aabc%2Fdef%2Bghi%3D%3D/disable")) {
		t.Fatalf("expected disable action to URL-encode fingerprint; got HTML: %s", html)
	}
}
