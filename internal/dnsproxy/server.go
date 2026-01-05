package dnsproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kompox/ssh-bastion/internal/config"
	bastiondns "github.com/kompox/ssh-bastion/internal/dns"
	"github.com/kompox/ssh-bastion/internal/storage"
	mdns "github.com/miekg/dns"
)

type Options struct {
	ListenAddr string
	Upstream   string
	Timeout    time.Duration
}

type Server struct {
	cfg      *config.Config
	registry *bastiondns.Registry

	upstream string
	client   *mdns.Client
	server   *mdns.Server
}

func New(opts Options) (*Server, error) {
	if strings.TrimSpace(opts.ListenAddr) == "" {
		return nil, errors.New("ListenAddr must be non-empty")
	}
	if strings.TrimSpace(opts.Upstream) == "" {
		return nil, errors.New("Upstream must be non-empty")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Second
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	store := storage.New(cfg.DataDir)
	registry := bastiondns.NewRegistry(store)

	s := &Server{
		cfg:      cfg,
		registry: registry,
		upstream: opts.Upstream,
		client:   &mdns.Client{Net: "udp", Timeout: opts.Timeout},
	}

	s.server = &mdns.Server{
		Addr:    opts.ListenAddr,
		Net:     "udp",
		Handler: mdns.HandlerFunc(s.handleDNS),
	}

	return s, nil
}

func (s *Server) LocalAddr() string {
	if s == nil || s.server == nil || s.server.PacketConn == nil {
		return ""
	}
	return s.server.PacketConn.LocalAddr().String()
}

func (s *Server) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		_ = s.server.Shutdown()
		err := <-errCh
		if err == nil || isServerClosedErr(err) {
			return nil
		}
		return err
	case err := <-errCh:
		if err == nil || isServerClosedErr(err) {
			return nil
		}
		return err
	}
}

func isServerClosedErr(err error) bool {
	if err == nil {
		return true
	}
	// miekg/dns does not expose a stable ErrServerClosed across versions.
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	// Fallback for older netpoll errors.
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

func (s *Server) handleDNS(w mdns.ResponseWriter, req *mdns.Msg) {
	resp, err := s.forward(req)
	if err != nil {
		m := new(mdns.Msg)
		m.SetRcode(req, mdns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	_ = w.WriteMsg(resp)
}

func (s *Server) forward(in *mdns.Msg) (*mdns.Msg, error) {
	if in == nil || len(in.Question) == 0 {
		return nil, errors.New("empty question")
	}
	if len(in.Question) != 1 {
		out, _, err := s.client.Exchange(in, s.upstream)
		return out, err
	}

	q := in.Question[0]
	// Aliases are intended to make SSH-style name resolution reliable, which relies
	// on A/AAAA. For other QTYPEs (e.g. MX, TXT), do not forward (to avoid upstream
	// resolver quirks) and do not claim NXDOMAIN (the name may exist for A/AAAA).
	// Return NODATA (NOERROR + empty answer) instead.
	if q.Qtype != mdns.TypeA && q.Qtype != mdns.TypeAAAA {
		m := new(mdns.Msg)
		m.SetReply(in)
		m.Rcode = mdns.RcodeSuccess
		return m, nil
	}
	origName := mdns.Fqdn(strings.TrimSuffix(q.Name, "."))

	aliases, err := s.registry.ListAliases()
	if err != nil {
		return nil, err
	}

	dest := ""
	for _, a := range aliases {
		src := mdns.Fqdn(strings.TrimSuffix(a.Source, "."))
		if strings.EqualFold(src, origName) {
			dest = a.Destination
			break
		}
	}

	if strings.TrimSpace(dest) == "" {
		if s.cfg != nil && s.cfg.DNSAliasesOnly {
			m := new(mdns.Msg)
			m.SetReply(in)
			m.Rcode = mdns.RcodeNameError
			return m, nil
		}
		out, _, err := s.client.Exchange(in, s.upstream)
		return out, err
	}

	rewrittenName := mdns.Fqdn(strings.TrimSuffix(dest, "."))
	outReq := in.Copy()
	outReq.Question[0].Name = rewrittenName

	out, _, err := s.client.Exchange(outReq, s.upstream)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("no response")
	}

	if len(out.Question) == 1 {
		out.Question[0].Name = origName
	}

	rewriteRRName := func(rr mdns.RR) {
		switch v := rr.(type) {
		case *mdns.A:
			if strings.EqualFold(v.Hdr.Name, rewrittenName) {
				v.Hdr.Name = origName
			}
		case *mdns.AAAA:
			if strings.EqualFold(v.Hdr.Name, rewrittenName) {
				v.Hdr.Name = origName
			}
		}
	}

	for _, rr := range out.Answer {
		rewriteRRName(rr)
	}
	for _, rr := range out.Ns {
		rewriteRRName(rr)
	}
	for _, rr := range out.Extra {
		rewriteRRName(rr)
	}

	return out, nil
}
