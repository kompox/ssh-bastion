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
	DataDir      string
	AuthMode     string
	UserIDHeader string
	EmailHeader  string
}

func Load() (*Config, error) {
	cfg := &Config{
		DataDir:  getEnv("SSHBASTION_DATA_DIR", "/data"),
		AuthMode: getEnv("SSHBASTION_AUTH_MODE", "easy_auth"),
	}

	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, fmt.Errorf("SSHBASTION_DATA_DIR must be non-empty")
	}

	switch cfg.AuthMode {
	case "easy_auth":
		cfg.UserIDHeader = getEnv("SSHBASTION_AUTH_USER_ID_HEADER", "X-MS-CLIENT-PRINCIPAL-ID")
		cfg.EmailHeader = getEnv("SSHBASTION_AUTH_EMAIL_HEADER", "X-MS-CLIENT-PRINCIPAL-NAME")
	case "oauth2_proxy":
		cfg.UserIDHeader = getEnv("SSHBASTION_AUTH_USER_ID_HEADER", "X-Auth-Request-User")
		cfg.EmailHeader = getEnv("SSHBASTION_AUTH_EMAIL_HEADER", "X-Auth-Request-Email")
	default:
		return nil, fmt.Errorf("invalid SSHBASTION_AUTH_MODE: %s (must be easy_auth or oauth2_proxy)", cfg.AuthMode)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
