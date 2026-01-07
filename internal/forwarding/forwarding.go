package forwarding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/kompox/ssh-bastion/internal/storage"
)

type Mode string

const (
	ModeAny    Mode = "any"
	ModeNone   Mode = "none"
	ModeCustom Mode = "custom"
)

func ParseMode(s string) (Mode, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch Mode(s) {
	case ModeAny, ModeNone, ModeCustom:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("invalid mode: %q", s)
	}
}

type Target struct {
	Rule    string `json:"rule"`
	Enabled bool   `json:"enabled"`
}

type Config struct {
	Mode              Mode     `json:"mode"`
	Targets           []Target `json:"targets"`
	RestartGeneration int64    `json:"restartGeneration"`
}

func DefaultConfig() Config {
	return Config{Mode: ModeAny, Targets: []Target{}, RestartGeneration: 0}
}

type Registry struct {
	store *storage.Store
}

func NewRegistry(store *storage.Store) *Registry {
	return &Registry{store: store}
}

const relPathForwardingJSON = "ssh/forwarding.json"

func (r *Registry) Load() (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, errors.New("storage is not configured")
	}

	path := r.store.Path(relPathForwardingJSON)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.Mode == "" {
		cfg.Mode = ModeAny
	}
	if cfg.Targets == nil {
		cfg.Targets = []Target{}
	}
	if _, err := ParseMode(string(cfg.Mode)); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (r *Registry) Save(cfg Config) error {
	if r == nil || r.store == nil {
		return errors.New("storage is not configured")
	}
	if cfg.Targets == nil {
		cfg.Targets = []Target{}
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return r.store.AtomicWrite(relPathForwardingJSON, b)
}

func (r *Registry) SetMode(mode Mode) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}
	parsed, err := ParseMode(string(mode))
	if err != nil {
		return err
	}
	cfg.Mode = parsed
	return r.Save(cfg)
}

func (r *Registry) AddTarget(rule string) error {
	rule = strings.TrimSpace(rule)
	if err := ValidateRule(rule); err != nil {
		return err
	}

	cfg, err := r.Load()
	if err != nil {
		return err
	}
	for _, t := range cfg.Targets {
		if t.Rule == rule {
			return fmt.Errorf("target already exists: %s", rule)
		}
	}
	cfg.Targets = append(cfg.Targets, Target{Rule: rule, Enabled: true})
	return r.Save(cfg)
}

func (r *Registry) UpdateTargetStatus(rule string, enabled bool) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("rule must be non-empty")
	}
	cfg, err := r.Load()
	if err != nil {
		return err
	}
	for i := range cfg.Targets {
		if cfg.Targets[i].Rule == rule {
			cfg.Targets[i].Enabled = enabled
			return r.Save(cfg)
		}
	}
	return os.ErrNotExist
}

func (r *Registry) DeleteTarget(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("rule must be non-empty")
	}
	cfg, err := r.Load()
	if err != nil {
		return err
	}

	found := false
	filtered := make([]Target, 0, len(cfg.Targets))
	for _, t := range cfg.Targets {
		if t.Rule == rule {
			found = true
			continue
		}
		filtered = append(filtered, t)
	}
	if !found {
		return os.ErrNotExist
	}
	cfg.Targets = filtered
	return r.Save(cfg)
}

func (r *Registry) EnabledRules() ([]string, error) {
	cfg, err := r.Load()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, t := range cfg.Targets {
		if t.Enabled {
			out = append(out, t.Rule)
		}
	}
	return out, nil
}

func (r *Registry) RestartGeneration() (int64, error) {
	cfg, err := r.Load()
	if err != nil {
		return 0, err
	}
	return cfg.RestartGeneration, nil
}

var ipv6InnerRe = regexp.MustCompile(`^[0-9a-fA-F:]+$`)

func ValidateRule(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("rule must be non-empty")
	}
	if len(rule) > 255 {
		return fmt.Errorf("rule must be no more than 255 characters")
	}
	if strings.ContainsRune(rule, '#') {
		return fmt.Errorf("rule must not contain '#'")
	}
	if strings.EqualFold(rule, "any") || strings.EqualFold(rule, "none") {
		return fmt.Errorf("reserved word is not a valid rule")
	}
	for _, r := range rule {
		if unicode.IsSpace(r) {
			return fmt.Errorf("rule must not contain whitespace")
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("rule must not contain control characters")
		}
	}

	host, port, err := splitHostPort(rule)
	if err != nil {
		return err
	}
	if err := validateHost(host); err != nil {
		return err
	}
	if err := validatePort(port); err != nil {
		return err
	}
	return nil
}

func splitHostPort(rule string) (string, string, error) {
	if strings.HasPrefix(rule, "[") {
		end := strings.Index(rule, "]")
		if end < 0 {
			return "", "", fmt.Errorf("invalid IPv6 rule: missing closing bracket")
		}
		if end+1 >= len(rule) || rule[end+1] != ':' {
			return "", "", fmt.Errorf("invalid IPv6 rule: expected ]:port")
		}
		host := rule[:end+1]
		port := rule[end+2:]
		if host == "[]" {
			return "", "", fmt.Errorf("IPv6 host must be non-empty")
		}
		if port == "" {
			return "", "", fmt.Errorf("port must be non-empty")
		}
		return host, port, nil
	}

	if strings.Count(rule, ":") != 1 {
		return "", "", fmt.Errorf("rule must be in host:port form")
	}
	parts := strings.SplitN(rule, ":", 2)
	host := parts[0]
	port := parts[1]
	if host == "" {
		return "", "", fmt.Errorf("host must be non-empty")
	}
	if port == "" {
		return "", "", fmt.Errorf("port must be non-empty")
	}
	return host, port, nil
}

func validateHost(host string) error {
	if host == "*" {
		return nil
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
		if inner == "" {
			return fmt.Errorf("IPv6 host must be non-empty")
		}
		if !ipv6InnerRe.MatchString(inner) {
			return fmt.Errorf("invalid IPv6 host")
		}
		ip := net.ParseIP(inner)
		if ip == nil || ip.To16() == nil || ip.To4() != nil {
			return fmt.Errorf("invalid IPv6 address")
		}
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil {
			return fmt.Errorf("IPv6 address must use bracket form: [addr]:port")
		}
		return nil
	}

	if len(host) > 253 {
		return fmt.Errorf("host must be no more than 253 characters")
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return fmt.Errorf("host must not start or end with a dot")
	}
	labels := strings.Split(host, ".")
	for i, label := range labels {
		if label == "" {
			return fmt.Errorf("host must not contain empty labels")
		}
		if label == "*" {
			if i != 0 {
				return fmt.Errorf("wildcard label '*' is only allowed as the first label")
			}
			continue
		}
		if len(label) > 63 {
			return fmt.Errorf("host label %q must be no more than 63 characters", label)
		}
		if !dnsLabelLike(label) {
			return fmt.Errorf("host label %q contains invalid characters", label)
		}
	}
	return nil
}

func dnsLabelLike(label string) bool {
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}
	return true
}

func validatePort(port string) error {
	if port == "*" {
		return nil
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port")
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}
	return nil
}
