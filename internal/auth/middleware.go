package auth

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kompox/ssh-bastion/internal/config"
)

type contextKey string

const (
	userIDKey  contextKey = "userID"
	emailKey   contextKey = "email"
	userDirKey contextKey = "userDir"
)

type Middleware struct {
	cfg *config.Config
}

func NewMiddleware(cfg *config.Config) *Middleware {
	return &Middleware{cfg: cfg}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID, email string

		if m.cfg.OverrideUserID != "" && m.cfg.OverrideEmail != "" {
			userID = m.cfg.OverrideUserID
			email = m.cfg.OverrideEmail
		} else {
			userID = strings.TrimSpace(r.Header.Get(m.cfg.UserIDHeader))
			email = strings.TrimSpace(r.Header.Get(m.cfg.EmailHeader))
		}

		if userID == "" || email == "" {
			writeUnauthorized(w, r, m.cfg.UserIDHeader, m.cfg.EmailHeader)
			return
		}

		userDirID := deriveUserDirID(userID)

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, emailKey, email)
		ctx = context.WithValue(ctx, userDirKey, userDirID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request, userIDHeader, emailHeader string) {
	accept := r.Header.Get("Accept")
	if accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Unauthorized</title></head><body><h1>Unauthorized</h1><p>Missing user identity headers.</p><ul><li>%s</li><li>%s</li></ul><p>For local testing, set both <code>SSHBASTION_AUTH_OVERRIDE_USER_ID</code> and <code>SSHBASTION_AUTH_OVERRIDE_EMAIL</code> (test mode only).</p></body></html>", userIDHeader, emailHeader)
		return
	}

	http.Error(w, "Unauthorized: missing user identity", http.StatusUnauthorized)
}

func deriveUserDirID(userID string) string {
	normalized := strings.ToLower(strings.TrimSpace(userID))
	namespaceUUID := uuid.Must(uuid.Parse(config.NamespaceUUID))
	return uuid.NewMD5(namespaceUUID, []byte(normalized)).String()
}

func GetUserID(r *http.Request) string {
	if v := r.Context().Value(userIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func GetEmail(r *http.Request) string {
	if v := r.Context().Value(emailKey); v != nil {
		return v.(string)
	}
	return ""
}

func GetUserDirID(r *http.Request) string {
	if v := r.Context().Value(userDirKey); v != nil {
		return v.(string)
	}
	return ""
}

func init() {
	_ = md5.New()
}
