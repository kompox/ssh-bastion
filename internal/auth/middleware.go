package auth

import (
	"context"
	"crypto/md5"
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
		userID := strings.TrimSpace(r.Header.Get(m.cfg.UserIDHeader))
		email := strings.TrimSpace(r.Header.Get(m.cfg.EmailHeader))

		if userID == "" || email == "" {
			http.Error(w, "Unauthorized: missing user identity", http.StatusUnauthorized)
			return
		}

		userDirID := deriveUserDirID(userID)

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, emailKey, email)
		ctx = context.WithValue(ctx, userDirKey, userDirID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
