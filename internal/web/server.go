package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kompox/ssh-bastion/internal/auth"
	"github.com/kompox/ssh-bastion/internal/config"
	"github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/keys"
	"github.com/kompox/ssh-bastion/internal/storage"
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

	mux.HandleFunc("GET /", srv.handleKeysPage)
	mux.HandleFunc("POST /keys", srv.handleAddKey)
	mux.HandleFunc("POST /keys/{fingerprint}/enable", srv.handleEnableKey)
	mux.HandleFunc("POST /keys/{fingerprint}/disable", srv.handleDisableKey)
	mux.HandleFunc("POST /keys/{fingerprint}/delete", srv.handleDeleteKey)

	mux.HandleFunc("GET /dns", srv.handleDNSPage)
	mux.HandleFunc("POST /dns", srv.handleAddAlias)
	mux.HandleFunc("POST /dns/{source}/delete", srv.handleDeleteAlias)

	handler := authMw.Authenticate(mux)

	log.Printf("Starting server on %s", addr)
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Auth mode: %s", cfg.AuthMode)
	if cfg.OverrideUserID != "" && cfg.OverrideEmail != "" {
		log.Printf("⚠️  TEST MODE: Using auth overrides (user: %s, email: %s)", cfg.OverrideUserID, cfg.OverrideEmail)
	}

	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleKeysPage(w http.ResponseWriter, r *http.Request) {
	s.renderKeysPage(w, r, http.StatusOK, "", "", "")
}

func (s *Server) renderKeysPage(w http.ResponseWriter, r *http.Request, status int, formPublicKey, flashKind, flashMsg string) {
	userDirID := auth.GetUserDirID(r)
	email := auth.GetEmail(r)
	if flashKind == "" {
		flashKind = "error"
	}

	keysList := []*keys.Key{}
	if s.keyRegistry != nil {
		var err error
		keysList, err = s.keyRegistry.ListKeys(userDirID)
		if err != nil {
			log.Printf("Error listing keys: %v", err)
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
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleAddKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)

	if err := r.ParseForm(); err != nil {
		s.renderKeysPage(w, r, http.StatusBadRequest, "", "error", "Invalid form")
		return
	}

	publicKey := r.FormValue("publicKey")
	if _, err := s.keyRegistry.AddKey(userDirID, publicKey); err != nil {
		s.renderKeysPage(w, r, http.StatusBadRequest, publicKey, "error", fmt.Sprintf("Failed to add key: %v", err))
		return
	}

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		log.Printf("Error rendering authorized_keys: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleEnableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
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
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		log.Printf("Error rendering authorized_keys: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDisableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
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
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		log.Printf("Error rendering authorized_keys: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
	fingerprint, err := url.PathUnescape(r.PathValue("fingerprint"))
	if err != nil {
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
		}
		s.renderKeysPage(w, r, status, "", kind, msg)
		return
	}

	if err := s.keyRegistry.RenderAuthorizedKeys(); err != nil {
		log.Printf("Error rendering authorized_keys: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDNSPage(w http.ResponseWriter, r *http.Request) {
	s.renderDNSPage(w, r, http.StatusOK, "", "", "", "")
}

func (s *Server) renderDNSPage(w http.ResponseWriter, r *http.Request, status int, formSource, formDestination, flashKind, flashMsg string) {
	email := auth.GetEmail(r)
	if flashKind == "" {
		flashKind = "error"
	}

	aliases := []dns.Alias{}
	if s.dnsRegistry != nil {
		var err error
		aliases, err = s.dnsRegistry.ListAliases()
		if err != nil {
			log.Printf("Error listing aliases: %v", err)
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
		"Aliases":         aliases,
		"Page":            "dns",
		"FlashKind":       flashKind,
		"FlashMessage":    flashMsg,
		"FormSource":      formSource,
		"FormDestination": formDestination,
	}

	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleAddAlias(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid form")
		return
	}

	source := r.FormValue("source")
	destination := r.FormValue("destination")

	if err := s.dnsRegistry.AddAlias(source, destination); err != nil {
		s.renderDNSPage(w, r, http.StatusBadRequest, source, destination, "error", fmt.Sprintf("Failed to add alias: %v", err))
		return
	}

	if err := s.dnsRegistry.RenderDnsmasqConf(); err != nil {
		log.Printf("Error rendering dnsmasq config: %v", err)
	}

	http.Redirect(w, r, "/dns", http.StatusSeeOther)
}

func (s *Server) handleDeleteAlias(w http.ResponseWriter, r *http.Request) {
	// Note: r.PathValue("source") may already be unescaped. If the decoded value contains
	// a literal '%', calling url.PathUnescape again would fail. Decode from EscapedPath
	// so we unescape exactly once.
	escapedPath := r.URL.EscapedPath()
	const prefix = "/dns/"
	const suffix = "/delete"
	if !strings.HasPrefix(escapedPath, prefix) || !strings.HasSuffix(escapedPath, suffix) {
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
		return
	}
	escapedSource := strings.TrimSuffix(strings.TrimPrefix(escapedPath, prefix), suffix)
	source, err := url.PathUnescape(escapedSource)
	if err != nil || source == "" {
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
		return
	}

	if err := s.dnsRegistry.DeleteAlias(source); err != nil {
		if os.IsNotExist(err) {
			s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "warning", "Alias not found")
			return
		}
		s.renderDNSPage(w, r, http.StatusInternalServerError, "", "", "error", "Failed to delete alias")
		return
	}

	if err := s.dnsRegistry.RenderDnsmasqConf(); err != nil {
		log.Printf("Error rendering dnsmasq config: %v", err)
	}

	http.Redirect(w, r, "/dns", http.StatusSeeOther)
}
