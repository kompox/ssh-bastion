package web

import (
	"fmt"
	"log"
	"strings"

	"github.com/kompox/ssh-bastion/internal/config"
)

func (s *Server) logInfo(msg string, kv ...any) {
	if s == nil || s.cfg == nil || !s.cfg.LogLevel.Enabled(config.LogLevelInfo) {
		return
	}
	log.Printf("INFO %s%s", msg, formatKV(kv...))
}

func (s *Server) logWarn(msg string, kv ...any) {
	if s == nil || s.cfg == nil || !s.cfg.LogLevel.Enabled(config.LogLevelWarn) {
		return
	}
	log.Printf("WARN %s%s", msg, formatKV(kv...))
}

func (s *Server) logError(err error, msg string, kv ...any) {
	if s == nil || s.cfg == nil || !s.cfg.LogLevel.Enabled(config.LogLevelError) {
		return
	}
	if err != nil {
		kv = append(kv, "err", err)
	}
	log.Printf("ERROR %s%s", msg, formatKV(kv...))
}

func formatKV(kv ...any) string {
	if len(kv) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(" ")

	for i := 0; i < len(kv); i += 2 {
		if i > 0 {
			b.WriteString(" ")
		}

		key := "?"
		if i < len(kv) {
			if k, ok := kv[i].(string); ok && k != "" {
				key = k
			} else {
				key = fmt.Sprint(kv[i])
			}
		}

		var value any = ""
		if i+1 < len(kv) {
			value = kv[i+1]
		}

		b.WriteString(key)
		b.WriteString("=")
		switch v := value.(type) {
		case string:
			b.WriteString(fmt.Sprintf("%q", v))
		default:
			b.WriteString(fmt.Sprint(v))
		}
	}

	return b.String()
}
