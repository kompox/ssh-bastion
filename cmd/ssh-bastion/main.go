package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kompox/ssh-bastion/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ssh-bastion <command>")
		fmt.Fprintln(os.Stderr, "Commands: web")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "web":
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
