package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kompox/ssh-bastion/internal/dnsproxy"
	"github.com/kompox/ssh-bastion/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ssh-bastion <command>")
		fmt.Fprintln(os.Stderr, "Commands: serve, web")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "serve":
		runServe()
	case "web":
		// Back-compat alias for HTTP-only server.
		runWeb()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
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
	upstream := fs.String("dns-upstream", getEnvDefault("SSHBASTION_DNS_UPSTREAM", "127.0.0.11:53"), "Upstream DNS resolver")

	fs.Parse(os.Args[2:])

	if !*enableHTTP && !*enableDNS {
		log.Fatalf("At least one service must be enabled: -http and/or -dns")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	if *enableDNS {
		proxy, err := dnsproxy.New(dnsproxy.Options{
			ListenAddr: *dnsAddr,
			Upstream:   *upstream,
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
