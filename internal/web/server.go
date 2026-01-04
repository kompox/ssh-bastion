package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kompox/ssh-bastion/internal/auth"
	"github.com/kompox/ssh-bastion/internal/config"
	"github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/keys"
	"github.com/kompox/ssh-bastion/internal/storage"
	"github.com/yuin/goldmark"
)

type Server struct {
	cfg         *config.Config
	store       *storage.Store
	keyRegistry *keys.Registry
	dnsRegistry *dns.Registry
	authMw      *auth.Middleware
	templates   *template.Template
}

func Run(addr string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store := storage.New(cfg.DataDir)
	keyRegistry := keys.NewRegistry(store)
	dnsRegistry := dns.NewRegistry(store)
	authMw := auth.NewMiddleware(cfg)

	tmpl, err := template.ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	srv := &Server{
		cfg:         cfg,
		store:       store,
		keyRegistry: keyRegistry,
		dnsRegistry: dnsRegistry,
		authMw:      authMw,
		templates:   tmpl,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", srv.handleHome)
	mux.HandleFunc("GET /ssh", srv.handleKeysPage)
	mux.HandleFunc("GET /admin", srv.requireAdmin(srv.handleAdminPage))
	mux.HandleFunc("GET /admin/users", srv.requireAdmin(srv.handleAdminUsersPage))
	mux.HandleFunc("GET /admin/keys", srv.requireAdmin(srv.handleAdminKeysPage))
	mux.HandleFunc("GET /admin/dns", srv.requireAdmin(srv.handleAdminDNSPage))

	mux.HandleFunc("POST /ssh/keys", srv.handleAddKey)
	mux.HandleFunc("POST /ssh/keys/{fingerprint}/enable", srv.handleEnableKey)
	mux.HandleFunc("POST /ssh/keys/{fingerprint}/disable", srv.handleDisableKey)
	mux.HandleFunc("POST /ssh/keys/{fingerprint}/delete", srv.handleDeleteKey)

	mux.HandleFunc("POST /admin/dns", srv.requireAdmin(srv.handleAddAlias))
	mux.HandleFunc("POST /admin/dns/{source}/delete", srv.requireAdmin(srv.handleDeleteAlias))

	handler := authMw.Authenticate(mux)

	testMode := cfg.OverrideUserID != "" && cfg.OverrideEmail != ""
	srv.logInfo("web server starting",
		"addr", addr,
		"dataDir", cfg.DataDir,
		"authMode", cfg.AuthMode,
		"testMode", testMode,
	)

	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	email := auth.GetEmail(r)
	role := auth.GetRole(r)
	flashKind := ""
	flashMsg := ""

	homePath := filepath.Join(s.cfg.DataDir, "content", "pages", "home.md")
	mdBytes, err := os.ReadFile(homePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logError(err, "read home markdown failed",
				"op", "home_read",
				"method", r.Method,
				"path", r.URL.Path,
			)
			flashKind = "error"
			flashMsg = "Failed to read home"
		}
		mdBytes = nil
	}

	var rendered template.HTML
	if len(mdBytes) > 0 {
		var sb strings.Builder
		if err := goldmark.Convert(mdBytes, &sb); err != nil {
			s.logError(err, "render home markdown failed",
				"op", "home_render",
				"method", r.Method,
				"path", r.URL.Path,
			)
			flashKind = "error"
			flashMsg = "Failed to render home"
		} else {
			rendered = template.HTML(sb.String())
		}
	}

	data := map[string]any{
		"Title":        "Home",
		"Email":        email,
		"Role":         role,
		"Page":         "home",
		"FlashKind":    flashKind,
		"FlashMessage": flashMsg,
		"HomeHTML":     rendered,
	}

	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "home",
			"method", r.Method,
			"path", r.URL.Path,
			"status", http.StatusOK,
		)
	}
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	email := auth.GetEmail(r)
	role := auth.GetRole(r)
	data := map[string]any{
		"Title": "Admin",
		"Email": email,
		"Role":  role,
		"Page":  "admin",
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "admin",
			"method", r.Method,
			"path", r.URL.Path,
			"status", http.StatusOK,
		)
	}
}

func (s *Server) handleKeysPage(w http.ResponseWriter, r *http.Request) {
	s.renderKeysPage(w, r, http.StatusOK, "", "", "")
}

func (s *Server) renderKeysPage(w http.ResponseWriter, r *http.Request, status int, formPublicKey, flashKind, flashMsg string) {
	userDirID := auth.GetUserDirID(r)
	userID := auth.GetUserID(r)
	email := auth.GetEmail(r)
	role := auth.GetRole(r)
	s.ensureUserProfile(userDirID, userID, email)
	if flashKind == "" {
		flashKind = "error"
	}

	keysList := []*keys.Key{}
	if s.keyRegistry != nil {
		var err error
		keysList, err = s.keyRegistry.ListKeys(userDirID)
		if err != nil {
			s.logError(err, "list keys failed",
				"op", "keys_list",
				"userDirID", userDirID,
				"method", r.Method,
				"path", r.URL.Path,
			)
			if flashMsg == "" {
				flashMsg = "Failed to list keys"
			}
			keysList = []*keys.Key{}
			if status == http.StatusOK {
				status = http.StatusInternalServerError
			}
		}
	}

	data := map[string]interface{}{
		"Title":         "SSH Keys",
		"Email":         email,
		"Role":          role,
		"Keys":          keysList,
		"Page":          "keys",
		"FlashKind":     flashKind,
		"FlashMessage":  flashMsg,
		"FormPublicKey": formPublicKey,
	}

	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "keys",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
		)
	}
}

func (s *Server) handleAddKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	userID := auth.GetUserID(r)
	email := auth.GetEmail(r)
	s.ensureUserProfile(userDirID, userID, email)

	if err := r.ParseForm(); err != nil {
		s.logWarn("parse form failed",
			"op", "key_add",
			"userDirID", userDirID,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderKeysPage(w, r, http.StatusBadRequest, "", "error", "Invalid form")
		return
	}

	publicKey := r.FormValue("publicKey")
	key, err := s.keyRegistry.AddKey(userDirID, publicKey)
	if err != nil {
		s.logWarn("add key failed",
			"op", "key_add",
			"userDirID", userDirID,
			"method", r.Method,
			"path", r.URL.Path,
			"err", err,
		)
		s.renderKeysPage(w, r, http.StatusBadRequest, publicKey, "error", fmt.Sprintf("Failed to add key: %v", err))
		return
	}

	s.logInfo("key added",
		"op", "key_add",
		"userDirID", userDirID,
		"fingerprint", key.Fingerprint,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		s.logError(err, "render authorized_keys failed",
			"op", "authorized_keys_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	http.Redirect(w, r, "/ssh", http.StatusSeeOther)
}

func (s *Server) handleEnableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	userID := auth.GetUserID(r)
	email := auth.GetEmail(r)
	s.ensureUserProfile(userDirID, userID, email)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
		s.logWarn("invalid fingerprint",
			"op", "key_enable",
			"userDirID", userDirID,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderKeysPage(w, r, http.StatusBadRequest, "", "error", "Invalid fingerprint")
		return
	}

	if err := s.keyRegistry.UpdateKeyStatus(userDirID, fingerprint, true); err != nil {
		status := http.StatusInternalServerError
		msg := "Failed to enable key"
		kind := "error"
		if os.IsNotExist(err) {
			status = http.StatusBadRequest
			msg = "Key not found"
			kind = "warning"
			s.logWarn("enable key: not found",
				"op", "key_enable",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		} else {
			s.logError(err, "enable key failed",
				"op", "key_enable",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	s.logInfo("key enabled",
		"op", "key_enable",
		"userDirID", userDirID,
		"fingerprint", fingerprint,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		s.logError(err, "render authorized_keys failed",
			"op", "authorized_keys_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	http.Redirect(w, r, "/ssh", http.StatusSeeOther)
}

func (s *Server) handleDisableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	userID := auth.GetUserID(r)
	email := auth.GetEmail(r)
	s.ensureUserProfile(userDirID, userID, email)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
		s.logWarn("invalid fingerprint",
			"op", "key_disable",
			"userDirID", userDirID,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderKeysPage(w, r, http.StatusBadRequest, "", "error", "Invalid fingerprint")
		return
	}

	if err := s.keyRegistry.UpdateKeyStatus(userDirID, fingerprint, false); err != nil {
		status := http.StatusInternalServerError
		msg := "Failed to disable key"
		kind := "error"
		if os.IsNotExist(err) {
			status = http.StatusBadRequest
			msg = "Key not found"
			kind = "warning"
			s.logWarn("disable key: not found",
				"op", "key_disable",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		} else {
			s.logError(err, "disable key failed",
				"op", "key_disable",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	s.logInfo("key disabled",
		"op", "key_disable",
		"userDirID", userDirID,
		"fingerprint", fingerprint,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		s.logError(err, "render authorized_keys failed",
			"op", "authorized_keys_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	http.Redirect(w, r, "/ssh", http.StatusSeeOther)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	userID := auth.GetUserID(r)
	email := auth.GetEmail(r)
	s.ensureUserProfile(userDirID, userID, email)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
		s.logWarn("invalid fingerprint",
			"op", "key_delete",
			"userDirID", userDirID,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderKeysPage(w, r, http.StatusBadRequest, "", "error", "Invalid fingerprint")
		return
	}

	if err := s.keyRegistry.DeleteKey(userDirID, fingerprint); err != nil {
		status := http.StatusInternalServerError
		msg := "Failed to delete key"
		kind := "error"
		if os.IsNotExist(err) {
			status = http.StatusBadRequest
			msg = "Key not found"
			kind = "warning"
			s.logWarn("delete key: not found",
				"op", "key_delete",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		} else {
			s.logError(err, "delete key failed",
				"op", "key_delete",
				"userDirID", userDirID,
				"fingerprint", fingerprint,
				"method", r.Method,
				"path", r.URL.Path,
			)
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	s.logInfo("key deleted",
		"op", "key_delete",
		"userDirID", userDirID,
		"fingerprint", fingerprint,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		s.logError(err, "render authorized_keys failed",
			"op", "authorized_keys_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	http.Redirect(w, r, "/ssh", http.StatusSeeOther)
}

func (s *Server) handleAdminDNSPage(w http.ResponseWriter, r *http.Request) {
	s.renderAdminDNSPage(w, r, http.StatusOK, "", "", "", "")
}

func (s *Server) renderAdminDNSPage(w http.ResponseWriter, r *http.Request, status int, formSource, formDestination, flashKind, flashMsg string) {
	email := auth.GetEmail(r)
	role := auth.GetRole(r)
	if flashKind == "" {
		flashKind = "error"
	}

	aliases := []dns.Alias{}
	if s.dnsRegistry != nil {
		var err error
		aliases, err = s.dnsRegistry.ListAliases()
		if err != nil {
			s.logError(err, "list aliases failed",
				"op", "dns_list",
				"method", r.Method,
				"path", r.URL.Path,
			)
			if flashMsg == "" {
				flashMsg = "Failed to list aliases"
			}
			aliases = []dns.Alias{}
			if status == http.StatusOK {
				status = http.StatusInternalServerError
			}
		}
	}

	data := map[string]interface{}{
		"Title":           "DNS Aliases",
		"Email":           email,
		"Role":            role,
		"Aliases":         aliases,
		"Page":            "admin_dns",
		"FlashKind":       flashKind,
		"FlashMessage":    flashMsg,
		"FormSource":      formSource,
		"FormDestination": formDestination,
	}

	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "admin_dns",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
		)
	}
}

func (s *Server) handleAddAlias(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.logWarn("parse form failed",
			"op", "dns_add",
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderAdminDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid form")
		return
	}

	source := r.FormValue("source")
	destination := r.FormValue("destination")

	if err := s.dnsRegistry.AddAlias(source, destination); err != nil {
		s.logWarn("add alias failed",
			"op", "dns_add",
			"source", source,
			"destination", destination,
			"method", r.Method,
			"path", r.URL.Path,
			"err", err,
		)
		s.renderAdminDNSPage(w, r, http.StatusBadRequest, source, destination, "error", fmt.Sprintf("Failed to add alias: %v", err))
		return
	}

	s.logInfo("alias added",
		"op", "dns_add",
		"source", source,
		"destination", destination,
		"method", r.Method,
		"path", r.URL.Path,
	)

	http.Redirect(w, r, "/admin/dns", http.StatusSeeOther)
}

func (s *Server) handleDeleteAlias(w http.ResponseWriter, r *http.Request) {
	// Note: r.PathValue("source") may already be unescaped. If the decoded value contains
	// a literal '%', calling url.PathUnescape again would fail. Decode from EscapedPath
	// so we unescape exactly once.
	escapedPath := r.URL.EscapedPath()
	const prefix = "/admin/dns/"
	const suffix = "/delete"
	if !strings.HasPrefix(escapedPath, prefix) || !strings.HasSuffix(escapedPath, suffix) {
		s.logWarn("invalid delete alias path",
			"op", "dns_delete",
			"method", r.Method,
			"path", r.URL.Path,
			"escapedPath", escapedPath,
		)
		s.renderAdminDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
		return
	}
	escapedSource := strings.TrimSuffix(strings.TrimPrefix(escapedPath, prefix), suffix)
	source, err := url.PathUnescape(escapedSource)
	if err != nil || source == "" {
		s.logWarn("invalid alias source",
			"op", "dns_delete",
			"method", r.Method,
			"path", r.URL.Path,
			"escapedSource", escapedSource,
		)
		s.renderAdminDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
		return
	}

	if err := s.dnsRegistry.DeleteAlias(source); err != nil {
		if os.IsNotExist(err) {
			s.logWarn("delete alias: not found",
				"op", "dns_delete",
				"source", source,
				"method", r.Method,
				"path", r.URL.Path,
			)
			s.renderAdminDNSPage(w, r, http.StatusBadRequest, "", "", "warning", "Alias not found")
			return
		}
		s.logError(err, "delete alias failed",
			"op", "dns_delete",
			"source", source,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderAdminDNSPage(w, r, http.StatusInternalServerError, "", "", "error", "Failed to delete alias")
		return
	}

	s.logInfo("alias deleted",
		"op", "dns_delete",
		"source", source,
		"method", r.Method,
		"path", r.URL.Path,
	)

	http.Redirect(w, r, "/admin/dns", http.StatusSeeOther)
}

type userProfile struct {
	UserID    string    `json:"userID"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Server) ensureUserProfile(userDirID, userID, email string) {
	if s.store == nil {
		return
	}
	if strings.TrimSpace(userDirID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(email) == "" {
		return
	}

	p := userProfile{UserID: userID, Email: email, UpdatedAt: time.Now().UTC()}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	_ = s.store.AtomicWrite(filepath.Join("users", userDirID, "profile.json"), b)
}

type adminUserRow struct {
	Email    string
	UserID   string
	KeyCount int
}

func (s *Server) handleAdminUsersPage(w http.ResponseWriter, r *http.Request) {
	email := auth.GetEmail(r)
	role := auth.GetRole(r)

	rows, err := s.listAdminUsers(true)
	flashKind := ""
	flashMsg := ""
	status := http.StatusOK
	if err != nil {
		status = http.StatusInternalServerError
		flashKind = "error"
		flashMsg = "Failed to list users"
		s.logError(err, "list admin users failed",
			"op", "admin_users_list",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	data := map[string]any{
		"Title":        "Admin Users",
		"Email":        email,
		"Role":         role,
		"Page":         "admin_users",
		"Users":        rows,
		"FlashKind":    flashKind,
		"FlashMessage": flashMsg,
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "admin_users",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
		)
	}
}

func (s *Server) handleAdminKeysPage(w http.ResponseWriter, r *http.Request) {
	email := auth.GetEmail(r)
	role := auth.GetRole(r)

	rows, err := s.listAdminKeys()
	flashKind := ""
	flashMsg := ""
	status := http.StatusOK
	if err != nil {
		status = http.StatusInternalServerError
		flashKind = "error"
		flashMsg = "Failed to list keys"
		s.logError(err, "list admin keys failed",
			"op", "admin_keys_list",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	data := map[string]any{
		"Title":        "Admin Keys",
		"Email":        email,
		"Role":         role,
		"Page":         "admin_keys",
		"Keys":         rows,
		"FlashKind":    flashKind,
		"FlashMessage": flashMsg,
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "admin_keys",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
		)
	}
}

type adminKeyRow struct {
	OwnerEmail  string
	Fingerprint string
	Status      string
	CreatedAt   string
}

func (s *Server) listAdminKeys() ([]adminKeyRow, error) {
	if s.store == nil || s.keyRegistry == nil {
		return []adminKeyRow{}, nil
	}

	usersDir := s.store.Path("users")
	entries, err := os.ReadDir(usersDir)
	if os.IsNotExist(err) {
		return []adminKeyRow{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows := make([]adminKeyRow, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		userDirID := e.Name()

		ownerEmail := userDirID
		profilePath := s.store.Path(filepath.Join("users", userDirID, "profile.json"))
		if b, err := os.ReadFile(profilePath); err == nil {
			var p userProfile
			if err := json.Unmarshal(b, &p); err == nil {
				if strings.TrimSpace(p.Email) != "" {
					ownerEmail = p.Email
				}
			}
		}

		keysList, err := s.keyRegistry.ListKeys(userDirID)
		if err != nil {
			continue
		}
		for _, k := range keysList {
			status := "disabled"
			if k.Enabled {
				status = "enabled"
			}
			rows = append(rows, adminKeyRow{
				OwnerEmail:  ownerEmail,
				Fingerprint: k.Fingerprint,
				Status:      status,
				CreatedAt:   k.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return rows, nil
}

func (s *Server) listAdminUsers(onlyWithKeys bool) ([]adminUserRow, error) {
	if s.store == nil {
		return []adminUserRow{}, nil
	}

	usersDir := s.store.Path("users")
	entries, err := os.ReadDir(usersDir)
	if os.IsNotExist(err) {
		return []adminUserRow{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows := make([]adminUserRow, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		userDirID := e.Name()

		keyCount := 0
		if s.keyRegistry != nil {
			keysList, err := s.keyRegistry.ListKeys(userDirID)
			if err != nil {
				continue
			}
			keyCount = len(keysList)
		}
		if onlyWithKeys && keyCount == 0 {
			continue
		}

		profilePath := s.store.Path(filepath.Join("users", userDirID, "profile.json"))
		b, err := os.ReadFile(profilePath)
		if err != nil {
			continue
		}
		var p userProfile
		if err := json.Unmarshal(b, &p); err != nil {
			continue
		}
		rows = append(rows, adminUserRow{Email: p.Email, UserID: p.UserID, KeyCount: keyCount})
	}

	return rows, nil
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth.GetRole(r) != "admin" {
			writeForbidden(w, r)
			return
		}
		next(w, r)
	}
}

func writeForbidden(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Forbidden</title></head><body><h1>Forbidden</h1><p>You do not have access to this page.</p></body></html>")
		return
	}

	http.Error(w, "Forbidden", http.StatusForbidden)
}
