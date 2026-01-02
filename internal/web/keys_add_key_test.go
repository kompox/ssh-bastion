package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kompox/ssh-bastion/internal/auth"
	"github.com/kompox/ssh-bastion/internal/config"
	"github.com/kompox/ssh-bastion/internal/keys"
	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestAddKey_InvalidKey_RendersPageWithError(t *testing.T) {
	root := findRepoRoot(t)

	tmp := t.TempDir()
	store := storage.New(tmp)
	keyRegistry := keys.NewRegistry(store)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{
		keyRegistry: keyRegistry,
		templates:   tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /keys", srv.handleAddKey)

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y"}
	authMw := auth.NewMiddleware(cfg)
	handler := authMw.Authenticate(mux)

	invalidKey := "not-a-key"
	form := url.Values{}
	form.Set("publicKey", invalidKey)

	req := httptest.NewRequest("POST", "/keys", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>SSH Public Keys</h1>") {
		t.Fatalf("expected keys page HTML; got: %s", body)
	}
	if !strings.Contains(body, "Failed to add key") {
		t.Fatalf("expected inline error message; got: %s", body)
	}
	if !strings.Contains(body, invalidKey) {
		t.Fatalf("expected textarea to preserve input; got: %s", body)
	}
}
