package dns

import (
	"os"
	"strings"
	"testing"

	"github.com/kompox/ssh-bastion/internal/storage"
)

func TestAddAlias(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	err := registry.AddAlias("gitea.example.com", "gitea.gitea.svc.cluster.local")
	if err != nil {
		t.Fatalf("Failed to add alias: %v", err)
	}

	aliases, err := registry.ListAliases()
	if err != nil {
		t.Fatalf("Failed to list aliases: %v", err)
	}

	if len(aliases) != 1 {
		t.Errorf("Expected 1 alias, got %d", len(aliases))
	}

	if aliases[0].Source != "gitea.example.com" {
		t.Errorf("Expected source 'gitea.example.com', got %s", aliases[0].Source)
	}

	if aliases[0].Destination != "gitea.gitea.svc.cluster.local" {
		t.Errorf("Expected destination 'gitea.gitea.svc.cluster.local', got %s", aliases[0].Destination)
	}
}

func TestDuplicateAliasRejected(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	err := registry.AddAlias("gitea.example.com", "gitea.gitea.svc.cluster.local")
	if err != nil {
		t.Fatalf("Failed to add first alias: %v", err)
	}

	err = registry.AddAlias("gitea.example.com", "other.svc.cluster.local")
	if err == nil {
		t.Error("Expected error when adding duplicate alias")
	}
}

func TestDeleteAlias(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	registry.AddAlias("gitea.example.com", "gitea.gitea.svc.cluster.local")
	registry.AddAlias("gitlab.example.com", "gitlab.gitlab.svc.cluster.local")

	err := registry.DeleteAlias("gitea.example.com")
	if err != nil {
		t.Fatalf("Failed to delete alias: %v", err)
	}

	aliases, err := registry.ListAliases()
	if err != nil {
		t.Fatalf("Failed to list aliases: %v", err)
	}

	if len(aliases) != 1 {
		t.Errorf("Expected 1 alias after deletion, got %d", len(aliases))
	}

	if aliases[0].Source != "gitlab.example.com" {
		t.Errorf("Expected remaining alias to be gitlab.example.com, got %s", aliases[0].Source)
	}
}

func TestRenderDnsmasqConf(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	registry.AddAlias("gitea.example.com", "gitea.gitea.svc.cluster.local")
	registry.AddAlias("gitlab.example.com", "gitlab.gitlab.svc.cluster.local")

	err := registry.RenderDnsmasqConf()
	if err != nil {
		t.Fatalf("Failed to render dnsmasq config: %v", err)
	}

	confPath := store.Path("dns/dnsmasq.d/generated.conf")
	content, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("Failed to read generated config: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "cname=gitea.example.com,gitea.gitea.svc.cluster.local") {
		t.Error("Expected gitea cname directive in config")
	}

	if !strings.Contains(contentStr, "cname=gitlab.example.com,gitlab.gitlab.svc.cluster.local") {
		t.Error("Expected gitlab cname directive in config")
	}
}

func TestEmptyAliasRejected(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.New(tmpDir)
	registry := NewRegistry(store)

	err := registry.AddAlias("", "destination.com")
	if err == nil {
		t.Error("Expected error for empty source")
	}

	err = registry.AddAlias("source.com", "")
	if err == nil {
		t.Error("Expected error for empty destination")
	}
}
