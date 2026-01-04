//go:build e2e

package e2e_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestE2E_AliasIsPersisted(t *testing.T) {
	httpBase := getEnvDefault("SSHBASTION_E2E_HTTP_BASE", "http://localhost:8080")
	dataDir := getDataDir()

	waitHTTP200(t, httpBase+"/", 60*time.Second)

	source := "e2e.local"
	destination := "ssh-bastion"

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{}
	form.Set("source", source)
	form.Set("destination", destination)

	resp, err := client.PostForm(httpBase+"/admin/dns", form)
	if err != nil {
		t.Fatalf("POST /admin/dns failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected POST /admin/dns to return 303, got %d", resp.StatusCode)
	}

	aliasesJSON := filepath.Join(dataDir, "dns", "aliases.json")
	content := waitReadFileContains(t, aliasesJSON, source, 5*time.Second)
	if !strings.Contains(content, destination) {
		t.Fatalf("aliases.json missing destination %q", destination)
	}
}

func TestE2E_DNSResolvesAlias(t *testing.T) {
	dnsAddr := getEnvDefault("SSHBASTION_E2E_DNS_ADDR", "127.0.0.1:5353")

	source := "e2e.local"

	name := dns.Fqdn(source)

	client := &dns.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(15 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		msg := new(dns.Msg)
		msg.SetQuestion(name, dns.TypeA)
		msg.RecursionDesired = true

		resp, _, err := client.Exchange(msg, dnsAddr)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if resp.Rcode != dns.RcodeSuccess {
			lastErr = fmt.Errorf("rcode=%s", dns.RcodeToString[resp.Rcode])
			time.Sleep(250 * time.Millisecond)
			continue
		}

		for _, rr := range resp.Answer {
			if a, ok := rr.(*dns.A); ok && a.A != nil && strings.EqualFold(a.Hdr.Name, name) {
				return
			}
		}

		lastErr = fmt.Errorf("expected an A answer for %q", name)
		time.Sleep(250 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	t.Fatalf("dns query did not meet expectations: %v", lastErr)
}

func TestE2E_DeleteDnsAlias(t *testing.T) {
	httpBase := getEnvDefault("SSHBASTION_E2E_HTTP_BASE", "http://localhost:8080")
	dataDir := getDataDir()

	source := "e2e.local"

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Delete the alias via the web API.
	deleteURL := fmt.Sprintf("%s/admin/dns/%s/delete", httpBase, url.PathEscape(source))
	resp, err := client.Post(deleteURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST /admin/dns/{source}/delete failed: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected delete to return 303, got %d", resp.StatusCode)
	}

	aliasesJSON := filepath.Join(dataDir, "dns", "aliases.json")
	_ = waitReadFileNotContains(t, aliasesJSON, source, 10*time.Second)
}

func TestE2E_DNSDoesNotResolveAlias(t *testing.T) {
	dnsAddr := getEnvDefault("SSHBASTION_E2E_DNS_ADDR", "127.0.0.1:5353")

	source := "e2e.local"

	name := dns.Fqdn(source)

	udpClient := &dns.Client{Timeout: 2 * time.Second, Net: "udp"}
	tcpClient := &dns.Client{Timeout: 2 * time.Second, Net: "tcp"}
	deadline := time.Now().Add(15 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		msg := new(dns.Msg)
		msg.SetQuestion(name, dns.TypeA)
		// Disable recursion so we only validate locally configured answers.
		// This avoids depending on upstream DNS behavior for non-existent names.
		msg.RecursionDesired = false

		retryOverTCP := false
		resp, _, err := udpClient.Exchange(msg, dnsAddr)
		if err != nil {
			// Some DNS servers can occasionally return UDP responses that are truncated
			// mid-record (or otherwise unparsable). Retry over TCP for stability.
			if strings.Contains(err.Error(), "overflow unpacking") {
				retryOverTCP = true
			} else {
				lastErr = err
				time.Sleep(250 * time.Millisecond)
				continue
			}
		}
		if resp != nil && resp.Truncated {
			retryOverTCP = true
		}
		if retryOverTCP {
			resp, _, err = tcpClient.Exchange(msg, dnsAddr)
			if err != nil {
				lastErr = err
				time.Sleep(250 * time.Millisecond)
				continue
			}
		}

		hasAnyAddress := false
		for _, rr := range resp.Answer {
			switch v := rr.(type) {
			case *dns.A:
				if v.A != nil && strings.EqualFold(v.Hdr.Name, name) {
					hasAnyAddress = true
				}
			case *dns.AAAA:
				if v.AAAA != nil && strings.EqualFold(v.Hdr.Name, name) {
					hasAnyAddress = true
				}
			}
		}

		// If the alias is deleted, the DNS proxy should no longer return an address.
		if !hasAnyAddress {
			return
		}

		// Include rcode for debugging; upstream failures are fine, but local answers are not.
		rcode := dns.RcodeToString[resp.Rcode]
		if rcode == "" {
			rcode = fmt.Sprintf("%d", resp.Rcode)
		}
		lastErr = fmt.Errorf("still returns an address after delete: rcode=%s", rcode)
		time.Sleep(250 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	t.Fatalf("dns still resolves deleted alias: %v", lastErr)
}

func waitHTTP200(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
		lastErr = fmt.Errorf("status=%d", resp.StatusCode)
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	t.Fatalf("http not ready: %v", lastErr)
}

func waitReadFileContains(t *testing.T, path string, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		content := string(b)
		if strings.Contains(content, substr) {
			return content
		}
		lastErr = fmt.Errorf("missing %q", substr)
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	t.Fatalf("file %s not ready: %v", path, lastErr)
	return ""
}

func waitReadFileNotContains(t *testing.T, path string, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		content := string(b)
		if !strings.Contains(content, substr) {
			return content
		}
		lastErr = fmt.Errorf("still contains %q", substr)
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	t.Fatalf("file %s did not update: %v", path, lastErr)
	return ""
}

func getDataDir() string {
	if v := os.Getenv("SSHBASTION_E2E_DATA_DIR"); v != "" {
		return v
	}
	// Tests run with package working dir typically at ./e2e.
	return filepath.Clean(filepath.Join("..", "_tmp", "data"))
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
