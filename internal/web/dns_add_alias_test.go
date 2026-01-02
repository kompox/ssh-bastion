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
	"github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestAddAlias_DuplicateSource_RendersPageWithError(t *testing.T) {
	root := findRepoRoot(t)

	tmp := t.TempDir()
	store := storage.New(tmp)
	dnsRegistry := dns.NewRegistry(store)

	if err := dnsRegistry.AddAlias("gitea.example.com", "gitea.gitea.svc.cluster.local"); err != nil {
		t.Fatalf("seed alias: %v", err)
	}

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{
		dnsRegistry: dnsRegistry,
		templates:   tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /dns", srv.handleAddAlias)

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y"}
	authMw := auth.NewMiddleware(cfg)
	handler := authMw.Authenticate(mux)

	form := url.Values{}
	form.Set("source", "gitea.example.com")
	form.Set("destination", "other.example.com")

	req := httptest.NewRequest("POST", "/dns", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>DNS Aliases</h1>") {
		t.Fatalf("expected DNS page HTML; got: %s", body)
	}
	if !strings.Contains(body, "class=\"flash flash-error\"") {
		t.Fatalf("expected flash error banner; got: %s", body)
	}
	if !strings.Contains(body, "Failed to add alias") {
		t.Fatalf("expected inline error message; got: %s", body)
	}
	if !strings.Contains(body, "value=\"gitea.example.com\"") {
		t.Fatalf("expected source value to be preserved; got: %s", body)
	}
	if !strings.Contains(body, "value=\"other.example.com\"") {
		t.Fatalf("expected destination value to be preserved; got: %s", body)
	}
}

func TestAddAlias_InvalidDNS1123_RendersPageWithError(t *testing.T) {
	root := findRepoRoot(t)

	tmp := t.TempDir()
	store := storage.New(tmp)
	dnsRegistry := dns.NewRegistry(store)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	srv := &Server{
		dnsRegistry: dnsRegistry,
		templates:   tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /dns", srv.handleAddAlias)

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y"}
	authMw := auth.NewMiddleware(cfg)
	handler := authMw.Authenticate(mux)

	form := url.Values{}
	form.Set("source", "Bad_Name.example.com")
	form.Set("destination", "dest.example.com")

	req := httptest.NewRequest("POST", "/dns", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>DNS Aliases</h1>") {
		t.Fatalf("expected DNS page HTML; got: %s", body)
	}
	if !strings.Contains(body, "class=\"flash flash-error\"") {
		t.Fatalf("expected flash error banner; got: %s", body)
	}
	if !strings.Contains(body, "Failed to add alias") {
		t.Fatalf("expected inline error message; got: %s", body)
	}
	if !strings.Contains(body, "value=\"Bad_Name.example.com\"") {
		t.Fatalf("expected source value to be preserved; got: %s", body)
	}
	if !strings.Contains(body, "value=\"dest.example.com\"") {
		t.Fatalf("expected destination value to be preserved; got: %s", body)
	}
}
