package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kompox/ssh-bastion/internal/config"
)

func TestAuthenticate_MissingHeaders_RendersHTMLUnauthorized(t *testing.T) {
	cfg := &config.Config{
		UserIDHeader: "X-User-ID",
		EmailHeader:  "X-User-Email",
	}

	mw := NewMiddleware(cfg)
	h := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://example.invalid/", nil)
	req.Header.Set("Accept", "text/html")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401; got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "<h1>Unauthorized</h1>") {
		t.Fatalf("expected HTML unauthorized page; got: %s", body)
	}
	if !strings.Contains(body, "X-User-ID") || !strings.Contains(body, "X-User-Email") {
		t.Fatalf("expected header names to be included; got: %s", body)
	}
}
