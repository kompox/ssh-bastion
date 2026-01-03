package web

import (
	"fmt"
	"html/template"
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

	testMode := cfg.OverrideUserID != "" && cfg.OverrideEmail != ""
	srv.logInfo("web server starting",
		"addr", addr,
		"dataDir", cfg.DataDir,
		"authMode", cfg.AuthMode,
		"testMode", testMode,
	)

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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleEnableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDisableKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	userDirID := auth.GetUserDirID(r)
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
		s.logError(err, "template execution failed",
			"op", "template_execute",
			"template", "layout.html",
			"page", "dns",
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
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid form")
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
		s.renderDNSPage(w, r, http.StatusBadRequest, source, destination, "error", fmt.Sprintf("Failed to add alias: %v", err))
		return
	}

	s.logInfo("alias added",
		"op", "dns_add",
		"source", source,
		"destination", destination,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.dnsRegistry.RenderDnsmasqConf(); err != nil {
		s.logError(err, "render dnsmasq config failed",
			"op", "dnsmasq_conf_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
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
		s.logWarn("invalid delete alias path",
			"op", "dns_delete",
			"method", r.Method,
			"path", r.URL.Path,
			"escapedPath", escapedPath,
		)
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
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
		s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "error", "Invalid source")
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
			s.renderDNSPage(w, r, http.StatusBadRequest, "", "", "warning", "Alias not found")
			return
		}
		s.logError(err, "delete alias failed",
			"op", "dns_delete",
			"source", source,
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.renderDNSPage(w, r, http.StatusInternalServerError, "", "", "error", "Failed to delete alias")
		return
	}

	s.logInfo("alias deleted",
		"op", "dns_delete",
		"source", source,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := s.dnsRegistry.RenderDnsmasqConf(); err != nil {
		s.logError(err, "render dnsmasq config failed",
			"op", "dnsmasq_conf_render",
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	http.Redirect(w, r, "/dns", http.StatusSeeOther)
}
