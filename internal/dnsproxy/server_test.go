package dnsproxy

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/storage"
	mdns "github.com/miekg/dns"
)

func TestRewriteARecordName(t *testing.T) {
	// Seed an alias in a temp data dir.
	dataDir := t.TempDir()
	os.Setenv("SSHBASTION_DATA_DIR", dataDir)
	t.Cleanup(func() { _ = os.Unsetenv("SSHBASTION_DATA_DIR") })

	store := storage.New(dataDir)
	reg := dns.NewRegistry(store)
	if err := reg.AddAlias("hoge.local", "example.com"); err != nil {
		t.Fatalf("add alias: %v", err)
	}

	// Start an upstream DNS server that returns an A record for example.com.
	upstreamAddr, stopUpstream := startUDPUpstream(t, func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		if len(r.Question) == 1 {
			q := r.Question[0]
			if q.Qtype == mdns.TypeA {
				m.Answer = append(m.Answer, &mdns.A{
					Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 30},
					A:   net.ParseIP("203.0.113.10").To4(),
				})
			}
		}
		_ = w.WriteMsg(m)
	})
	t.Cleanup(stopUpstream)

	proxy, err := New(Options{ListenAddr: "127.0.0.1:0", Upstream: upstreamAddr, Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- proxy.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = <-errCh
	})

	addr := waitForLocalAddr(t, proxy)

	// Query the proxy.
	c := &mdns.Client{Net: "udp", Timeout: 1 * time.Second}
	msg := new(mdns.Msg)
	msg.SetQuestion(mdns.Fqdn("hoge.local"), mdns.TypeA)

	resp, _, err := c.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if resp == nil {
		t.Fatalf("nil response")
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}
	a, ok := resp.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A, got %T", resp.Answer[0])
	}
	if a.Hdr.Name != mdns.Fqdn("hoge.local") {
		t.Fatalf("expected owner name rewritten to hoge.local, got %q", a.Hdr.Name)
	}
}

func TestDoesNotRewriteNonAddressQueries(t *testing.T) {
	// Seed an alias in a temp data dir.
	dataDir := t.TempDir()
	os.Setenv("SSHBASTION_DATA_DIR", dataDir)
	t.Cleanup(func() { _ = os.Unsetenv("SSHBASTION_DATA_DIR") })

	store := storage.New(dataDir)
	reg := dns.NewRegistry(store)
	if err := reg.AddAlias("hoge.local", "github.com"); err != nil {
		t.Fatalf("add alias: %v", err)
	}

	// Upstream: should not be used for MX queries.
	upstreamAddr, stopUpstream := startUDPUpstream(t, func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetRcode(r, mdns.RcodeServerFailure)
		_ = w.WriteMsg(m)
	})
	t.Cleanup(stopUpstream)

	proxy, err := New(Options{ListenAddr: "127.0.0.1:0", Upstream: upstreamAddr, Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- proxy.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = <-errCh
	})

	addr := waitForLocalAddr(t, proxy)

	// Query MX for the alias. We expect the proxy to block non-address QTYPEs and return
	// NODATA (NOERROR + empty answer).
	c := &mdns.Client{Net: "udp", Timeout: 1 * time.Second}
	msg := new(mdns.Msg)
	msg.SetQuestion(mdns.Fqdn("hoge.local"), mdns.TypeMX)

	resp, _, err := c.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if resp == nil {
		t.Fatalf("nil response")
	}
	if resp.Rcode != mdns.RcodeSuccess {
		t.Fatalf("expected NOERROR for MX query, got rcode=%d", resp.Rcode)
	}
	if len(resp.Answer) != 0 {
		t.Fatalf("expected no answers for MX query, got %d", len(resp.Answer))
	}
}

func TestAliasesOnly_ReturnsNXDOMAINForNonAliasAQuery(t *testing.T) {
	dataDir := t.TempDir()
	os.Setenv("SSHBASTION_DATA_DIR", dataDir)
	os.Setenv("SSHBASTION_DNS_ALIASES_ONLY", "true")
	t.Cleanup(func() {
		_ = os.Unsetenv("SSHBASTION_DATA_DIR")
		_ = os.Unsetenv("SSHBASTION_DNS_ALIASES_ONLY")
	})

	store := storage.New(dataDir)
	reg := dns.NewRegistry(store)
	if err := reg.AddAlias("hoge.local", "example.com"); err != nil {
		t.Fatalf("add alias: %v", err)
	}

	// Upstream: should not be used for non-aliased queries in aliases-only mode.
	upstreamAddr, stopUpstream := startUDPUpstream(t, func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetRcode(r, mdns.RcodeServerFailure)
		_ = w.WriteMsg(m)
	})
	t.Cleanup(stopUpstream)

	proxy, err := New(Options{ListenAddr: "127.0.0.1:0", Upstream: upstreamAddr, Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- proxy.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = <-errCh
	})

	addr := waitForLocalAddr(t, proxy)

	c := &mdns.Client{Net: "udp", Timeout: 1 * time.Second}
	msg := new(mdns.Msg)
	msg.SetQuestion(mdns.Fqdn("not-alias.local"), mdns.TypeA)

	resp, _, err := c.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if resp == nil {
		t.Fatalf("nil response")
	}
	if resp.Rcode != mdns.RcodeNameError {
		t.Fatalf("expected NXDOMAIN for non-aliased A query, got rcode=%d", resp.Rcode)
	}
}

func waitForLocalAddr(t *testing.T, s *Server) string {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if addr := s.LocalAddr(); addr != "" {
			return addr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("proxy did not bind to a local address")
	return "" // unreachable
}

func startUDPUpstream(t *testing.T, handler func(mdns.ResponseWriter, *mdns.Msg)) (addr string, stop func()) {
	t.Helper()

	s := &mdns.Server{Addr: "127.0.0.1:0", Net: "udp", Handler: mdns.HandlerFunc(handler)}

	errCh := make(chan error, 1)
	go func() { errCh <- s.ListenAndServe() }()

	// Wait for PacketConn to be ready.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.PacketConn != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.PacketConn == nil {
		t.Fatalf("upstream did not start")
	}

	return s.PacketConn.LocalAddr().String(), func() {
		_ = s.Shutdown()
		<-errCh
	}
}

func TestDoesNotCreateDnsmasqArtifacts(t *testing.T) {
	// Ensure the proxy does not require any dnsmasq-specific directories.
	dataDir := t.TempDir()
	os.Setenv("SSHBASTION_DATA_DIR", dataDir)
	t.Cleanup(func() { _ = os.Unsetenv("SSHBASTION_DATA_DIR") })

	_, err := New(Options{ListenAddr: "127.0.0.1:0", Upstream: "127.0.0.11:53"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "dns", "dnsmasq.d")); err == nil {
		t.Fatalf("unexpected dnsmasq.d directory")
	}
}
