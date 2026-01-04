package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectUpstreamFromResolvConf_IPv4(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "resolv.conf")
	if err := os.WriteFile(p, []byte("nameserver 1.1.1.1\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := detectUpstreamFromResolvConf(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "1.1.1.1:53"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDetectUpstreamFromResolvConf_IPv6(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "resolv.conf")
	if err := os.WriteFile(p, []byte("nameserver fd00::1\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := detectUpstreamFromResolvConf(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "[fd00::1]:53"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDetectUpstreamFromResolvConf_Empty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "resolv.conf")
	if err := os.WriteFile(p, []byte("# empty\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := detectUpstreamFromResolvConf(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestResolveUpstream_PrefersFlag(t *testing.T) {
	t.Parallel()

	old := os.Getenv("SSHBASTION_DNS_UPSTREAM")
	t.Cleanup(func() { _ = os.Setenv("SSHBASTION_DNS_UPSTREAM", old) })
	_ = os.Setenv("SSHBASTION_DNS_UPSTREAM", "9.9.9.9:53")

	got, err := resolveUpstream("8.8.8.8:53")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.8.8.8:53" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveUpstream_UsesEnvWhenFlagEmpty(t *testing.T) {
	t.Parallel()

	old := os.Getenv("SSHBASTION_DNS_UPSTREAM")
	t.Cleanup(func() { _ = os.Setenv("SSHBASTION_DNS_UPSTREAM", old) })
	_ = os.Setenv("SSHBASTION_DNS_UPSTREAM", "9.9.9.9:53")

	got, err := resolveUpstream("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "9.9.9.9:53" {
		t.Fatalf("got %q", got)
	}
}
