package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kompox/ssh-bastion/internal/auth"
	"github.com/kompox/ssh-bastion/internal/config"
	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestAdminHome_GET_RendersExistingMarkdown(t *testing.T) {
	root := findRepoRoot(t)

	dataDir := t.TempDir()
	store := storage.New(dataDir)
	if err := store.AtomicWrite(filepath.Join("content", "pages", "home.md"), []byte("# Hello\n")); err != nil {
		t.Fatalf("seed home.md: %v", err)
	}

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{cfg: &config.Config{DataDir: dataDir}, store: store, templates: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/home", srv.requireAdmin(srv.handleAdminHomePage))

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y", RoleDefault: "admin", RoleAdminIDs: map[string]struct{}{}}
	authMw := auth.NewMiddleware(cfg)
	h := authMw.Authenticate(mux)

	req := httptest.NewRequest("GET", "/admin/home", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>Home Page</h1>") {
		t.Fatalf("expected admin home page; got: %s", body)
	}
	if !strings.Contains(body, "# Hello") {
		t.Fatalf("expected markdown content to appear in textarea; got: %s", body)
	}
}

func TestAdminHome_POST_SavesAndRedirects(t *testing.T) {
	root := findRepoRoot(t)

	dataDir := t.TempDir()
	store := storage.New(dataDir)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{cfg: &config.Config{DataDir: dataDir}, store: store, templates: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/home", srv.requireAdmin(srv.handleAdminHomeSave))

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y", RoleDefault: "admin", RoleAdminIDs: map[string]struct{}{}}
	authMw := auth.NewMiddleware(cfg)
	h := authMw.Authenticate(mux)

	form := url.Values{}
	form.Set("markdown", "# Updated\n")

	req := httptest.NewRequest("POST", "/admin/home", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected 303; got %d", res.Code)
	}
	if loc := res.Header().Get("Location"); loc != "/admin/home?saved=1" {
		t.Fatalf("expected redirect to /admin/home?saved=1; got %q", loc)
	}

	b, err := os.ReadFile(filepath.Join(dataDir, "content", "pages", "home.md"))
	if err != nil {
		t.Fatalf("read saved home.md: %v", err)
	}
	if string(b) != "# Updated\n" {
		t.Fatalf("unexpected saved content: %q", string(b))
	}
}

func TestAdminHome_UserForbidden(t *testing.T) {
	root := findRepoRoot(t)

	dataDir := t.TempDir()
	store := storage.New(dataDir)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{cfg: &config.Config{DataDir: dataDir}, store: store, templates: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/home", srv.requireAdmin(srv.handleAdminHomePage))

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y", RoleDefault: "user", RoleAdminIDs: map[string]struct{}{}}
	authMw := auth.NewMiddleware(cfg)
	h := authMw.Authenticate(mux)

	req := httptest.NewRequest("GET", "/admin/home", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403; got %d", res.Code)
	}
}
