package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kompox/ssh-bastion/internal/auth"
	"github.com/kompox/ssh-bastion/internal/config"
	"github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestDeleteAlias_MissingAlias_RendersPageWithError(t *testing.T) {
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
	mux.HandleFunc("POST /dns/{source}/delete", srv.handleDeleteAlias)

	cfg := &config.Config{OverrideUserID: "test-user", OverrideEmail: "test@example.com", UserIDHeader: "X", EmailHeader: "Y"}
	authMw := auth.NewMiddleware(cfg)
	handler := authMw.Authenticate(mux)

	req := httptest.NewRequest("POST", "/dns/nonexists.example.com/delete", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>DNS Aliases</h1>") {
		t.Fatalf("expected DNS page HTML; got: %s", body)
	}
	if !strings.Contains(body, "class=\"flash flash-warning\"") {
		t.Fatalf("expected flash warning banner; got: %s", body)
	}
	if !strings.Contains(body, "Alias not found") {
		t.Fatalf("expected inline error message; got: %s", body)
	}
}
