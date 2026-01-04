package web

import (
	"bytes"
	"html/template"
	"path/filepath"
	"testing"

	"github.com/kompox/ssh-bastion/internal/dns"
)

func TestDnsTemplate_URLencodesSourceInDeleteAction(t *testing.T) {
	root := findRepoRoot(t)

	tmpl, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	// DNS-1123-valid source.
	source := "foo-bar.example.com"
	data := map[string]any{
		"Title":   "DNS Aliases",
		"Email":   "test@example.com",
		"Page":    "admin_dns",
		"Aliases": []dns.Alias{{Source: source, Destination: "dest.example.com"}},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	html := buf.String()
	if !bytes.Contains([]byte(html), []byte("/admin/dns/foo-bar.example.com/delete")) {
		t.Fatalf("expected delete action to include source; got HTML: %s", html)
	}
}
