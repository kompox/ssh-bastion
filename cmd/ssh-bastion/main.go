package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kompox/ssh-bastion/internal/dnsproxy"
	"github.com/kompox/ssh-bastion/internal/forwarding"
	"github.com/kompox/ssh-bastion/internal/storage"
	"github.com/kompox/ssh-bastion/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ssh-bastion <command>")
		fmt.Fprintln(os.Stderr, "Commands: serve, web, sshd-config, forwarding-restart-generation")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "serve":
		runServe()
	case "web":
		// Back-compat alias for HTTP-only server.
		runWeb()
	case "sshd-config":
		runSSHDConfig()
	case "forwarding-restart-generation":
		runForwardingRestartGeneration()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func runForwardingRestartGeneration() {
	fs := flag.NewFlagSet("forwarding-restart-generation", flag.ExitOnError)
	dataDir := fs.String("data-dir", getEnvDefault("SSHBASTION_DATA_DIR", "/data"), "Data directory (default: SSHBASTION_DATA_DIR or /data)")
	fs.Parse(os.Args[2:])

	store := storage.New(*dataDir)
	reg := forwarding.NewRegistry(store)
	gen, err := reg.RestartGeneration()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, gen)
}

func runSSHDConfig() {
	fs := flag.NewFlagSet("sshd-config", flag.ExitOnError)
	dataDir := fs.String("data-dir", getEnvDefault("SSHBASTION_DATA_DIR", "/data"), "Data directory (default: SSHBASTION_DATA_DIR or /data)")
	authorizedKeysFile := fs.String("authorized-keys-file", "", "AuthorizedKeysFile path")
	hostKeysDir := fs.String("host-keys-dir", "", "Host keys directory")
	logLevel := fs.String("log-level", getEnvDefault("SSHBASTION_SSHD_LOG_LEVEL", "INFO"), "sshd LogLevel")
	fs.Parse(os.Args[2:])

	if strings.TrimSpace(*authorizedKeysFile) == "" {
		fmt.Fprintln(os.Stderr, "-authorized-keys-file is required")
		os.Exit(2)
	}
	if strings.TrimSpace(*hostKeysDir) == "" {
		fmt.Fprintln(os.Stderr, "-host-keys-dir is required")
		os.Exit(2)
	}

	store := storage.New(*dataDir)
	reg := forwarding.NewRegistry(store)
	fwd, err := reg.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	permitOpen := ""
	switch fwd.Mode {
	case forwarding.ModeAny:
		permitOpen = ""
	case forwarding.ModeNone:
		permitOpen = "PermitOpen none\n"
	case forwarding.ModeCustom:
		enabled, err := reg.EnabledRules()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(enabled) == 0 {
			permitOpen = "PermitOpen none\n"
		} else {
			permitOpen = "PermitOpen " + strings.Join(enabled, " ") + "\n"
		}
	default:
		fmt.Fprintf(os.Stderr, "invalid forwarding mode: %q\n", fwd.Mode)
		os.Exit(1)
	}

	var sb strings.Builder
	sb.WriteString("Port 22\n")
	sb.WriteString("ListenAddress 0.0.0.0\n\n")
	sb.WriteString("HostKey " + strings.TrimRight(*hostKeysDir, "/") + "/ssh_host_ed25519_key\n")
	sb.WriteString("HostKey " + strings.TrimRight(*hostKeysDir, "/") + "/ssh_host_rsa_key\n\n")
	sb.WriteString("PasswordAuthentication no\n")
	sb.WriteString("KbdInteractiveAuthentication no\n")
	sb.WriteString("PermitRootLogin no\n")
	sb.WriteString("PubkeyAuthentication yes\n")
	sb.WriteString("AuthenticationMethods publickey\n\n")
	sb.WriteString("AuthorizedKeysFile " + *authorizedKeysFile + "\n\n")
	sb.WriteString("AllowTcpForwarding yes\n")
	if permitOpen != "" {
		sb.WriteString(permitOpen)
	}
	sb.WriteString("X11Forwarding no\n")
	sb.WriteString("AllowAgentForwarding no\n")
	sb.WriteString("PermitTTY yes\n\n")
	sb.WriteString("UseDNS no\n")
	sb.WriteString("PrintMotd no\n")
	sb.WriteString("LogLevel " + *logLevel + "\n")
	sb.WriteString("Subsystem sftp internal-sftp\n\n")
	sb.WriteString("# This SSH endpoint is intended for ProxyJump / -W (direct-tcpip) only.\n")
	sb.WriteString("# For session/exec requests, show a message and disconnect.\n")
	sb.WriteString("Match User jump\n")
	sb.WriteString("\tForceCommand /bin/sh /usr/local/bin/ssh-bastion-shell\n")

	fmt.Fprint(os.Stdout, sb.String())
}

func runWeb() {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	fs.Parse(os.Args[2:])

	if err := web.Run(*addr); err != nil {
		log.Fatalf("Error running web server: %v", err)
	}
}

func runServe() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	httpAddr := fs.String("http-addr", ":8080", "HTTP listen address")
	enableHTTP := fs.Bool("http", true, "Enable HTTP server")

	dnsAddr := fs.String("dns-addr", ":53", "DNS listen address (UDP)")
	enableDNS := fs.Bool("dns", false, "Enable DNS server")
	upstreamFlag := fs.String("dns-upstream", "", "Upstream DNS resolver (host:port); if unset, uses SSHBASTION_DNS_UPSTREAM or /etc/resolv.conf")

	fs.Parse(os.Args[2:])

	if !*enableHTTP && !*enableDNS {
		log.Fatalf("At least one service must be enabled: -http and/or -dns")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	if *enableDNS {
		upstream, err := resolveUpstream(*upstreamFlag)
		if err != nil {
			log.Fatalf("Error resolving DNS upstream: %v", err)
		}

		proxy, err := dnsproxy.New(dnsproxy.Options{
			ListenAddr: *dnsAddr,
			Upstream:   upstream,
			Timeout:    2 * time.Second,
		})
		if err != nil {
			log.Fatalf("Error starting DNS server: %v", err)
		}

		go func() {
			errCh <- proxy.Serve(ctx)
		}()
	}

	if *enableHTTP {
		go func() {
			errCh <- web.Run(*httpAddr)
		}()
	}

	// Exit on first error or signal.
	select {
	case <-ctx.Done():
		return
	case err := <-errCh:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func resolveUpstream(upstreamFlag string) (string, error) {
	if strings.TrimSpace(upstreamFlag) != "" {
		return upstreamFlag, nil
	}
	if v := strings.TrimSpace(os.Getenv("SSHBASTION_DNS_UPSTREAM")); v != "" {
		return v, nil
	}

	addr, err := detectUpstreamFromResolvConf("/etc/resolv.conf")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(addr) == "" {
		return "", fmt.Errorf("no upstream resolver configured (set -dns-upstream or SSHBASTION_DNS_UPSTREAM, or ensure /etc/resolv.conf contains a nameserver)")
	}
	return addr, nil
}

func detectUpstreamFromResolvConf(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "nameserver" {
			continue
		}
		ip := strings.TrimSpace(fields[1])
		if ip == "" {
			continue
		}
		parsed := net.ParseIP(ip)
		if parsed == nil {
			// Keep behavior strict; resolv.conf should contain an IP literal.
			return "", fmt.Errorf("invalid nameserver in %s: %q", path, ip)
		}
		if parsed.To4() != nil {
			return ip + ":53", nil
		}
		return "[" + ip + "]:53", nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return "", nil
}
