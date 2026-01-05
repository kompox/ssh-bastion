package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	NamespaceUUID = "9f7f2e3a-3c77-4d7b-a4d4-6d5167b1b5ad"
)

type Config struct {
	DataDir        string
	AuthMode       string
	UserIDHeader   string
	EmailHeader    string
	LogLevel       LogLevel
	OverrideUserID string
	OverrideEmail  string
	RoleDefault    string
	RoleAdminIDs   map[string]struct{}
	DNSAliasesOnly bool
}

type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

func ParseLogLevel(value string) (LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return LogLevelInfo, nil
	case "error":
		return LogLevelError, nil
	case "warn", "warning":
		return LogLevelWarn, nil
	case "debug":
		return LogLevelDebug, nil
	default:
		return 0, fmt.Errorf("invalid SSHBASTION_LOG_LEVEL: %q (must be error|warn|info|debug)", value)
	}
}

func (l LogLevel) Enabled(level LogLevel) bool {
	return l >= level
}

func Load() (*Config, error) {
	cfg := &Config{
		DataDir:  getEnv("SSHBASTION_DATA_DIR", "/data"),
		AuthMode: getEnv("SSHBASTION_AUTH_MODE", "easy_auth"),
	}

	logLevel, err := ParseLogLevel(getEnv("SSHBASTION_LOG_LEVEL", "info"))
	if err != nil {
		return nil, err
	}
	cfg.LogLevel = logLevel

	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, fmt.Errorf("SSHBASTION_DATA_DIR must be non-empty")
	}

	switch cfg.AuthMode {
	case "easy_auth":
		cfg.UserIDHeader = getEnv("SSHBASTION_AUTH_USER_ID_HEADER", "X-MS-CLIENT-PRINCIPAL-ID")
		cfg.EmailHeader = getEnv("SSHBASTION_AUTH_EMAIL_HEADER", "X-MS-CLIENT-PRINCIPAL-NAME")
	case "oauth2_proxy":
		cfg.UserIDHeader = getEnv("SSHBASTION_AUTH_USER_ID_HEADER", "X-Forwarded-User")
		cfg.EmailHeader = getEnv("SSHBASTION_AUTH_EMAIL_HEADER", "X-Forwarded-Email")
	default:
		return nil, fmt.Errorf("invalid SSHBASTION_AUTH_MODE: %s (must be easy_auth or oauth2_proxy)", cfg.AuthMode)
	}

	cfg.OverrideUserID = getEnv("SSHBASTION_AUTH_OVERRIDE_USER_ID", "")
	cfg.OverrideEmail = getEnv("SSHBASTION_AUTH_OVERRIDE_EMAIL", "")

	roleDefault := strings.ToLower(strings.TrimSpace(getEnv("SSHBASTION_ROLE_DEFAULT", "user")))
	switch roleDefault {
	case "user", "admin":
		cfg.RoleDefault = roleDefault
	default:
		return nil, fmt.Errorf("invalid SSHBASTION_ROLE_DEFAULT: %q (must be user|admin)", roleDefault)
	}

	cfg.RoleAdminIDs = make(map[string]struct{})
	for _, id := range splitCommaSeparated(getEnv("SSHBASTION_ROLE_ADMIN_IDS", "")) {
		cfg.RoleAdminIDs[id] = struct{}{}
	}

	cfg.DNSAliasesOnly = parseBoolEnv(getEnv("SSHBASTION_DNS_ALIASES_ONLY", ""))

	return cfg, nil
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

func splitCommaSeparated(value string) []string {
	items := strings.Split(value, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
