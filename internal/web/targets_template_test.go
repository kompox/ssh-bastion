package web

import (
	"bytes"
	"html/template"
	"path/filepath"
	"testing"

	"github.com/kompox/ssh-bastion/internal/forwarding"
)

func TestTargetsTemplate_URLencodesRuleInActions(t *testing.T) {
	root := findRepoRoot(t)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	rule := "db.example.com:5432"
	data := map[string]any{
		"Title":    "Forwarding Targets",
		"Email":    "test@example.com",
		"AuthMode": "oauth2_proxy",
		"Page":     "admin_targets",
		"Mode":     "custom",
		"Targets":  []forwarding.Target{{Rule: rule, Enabled: true}},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	html := buf.String()
	if !bytes.Contains([]byte(html), []byte("/admin/targets/db.example.com%3A5432/disable")) {
		t.Fatalf("expected action to include url-escaped rule; got HTML: %s", html)
	}
}
