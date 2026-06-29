package agentcli

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/ai-critic/client"
)

const pingHelp = `Usage: remote-agent ping

Ping the configured ai-critic server and report reachability.
Uses GET /ping (expects "pong"). Does not require authentication.

Uses the default server from saved config, or pass --server explicitly.

Exit code 0 means the server responded with pong; non-zero on failure.
`

func runPing(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(pingHelp)
			return nil
		}
		return fmt.Errorf("ping takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return fmt.Errorf("configuration error: %w\n\nHint: pass --server URL or run '%s config' to save a default domain", err, active.Name)
	}

	fmt.Printf("Server: %s\n", cli.Server)

	result, err := cli.Ping()
	if err != nil {
		fmt.Println()
		fmt.Print(formatPingFailure(cli.Server, err))
		return fmt.Errorf("ping failed")
	}

	fmt.Printf("Status: ok (pong in %s)\n", result.Latency.Round(time.Millisecond))
	return nil
}

func formatPingFailure(server string, err error) string {
	var buf strings.Builder
	buf.WriteString("Ping failed\n")
	buf.WriteString(fmt.Sprintf("  server: %s\n", server))
	buf.WriteString(fmt.Sprintf("  error:  %v\n", err))

	hints := pingFailureHints(server, err)
	buf.WriteString("\nSuggestions:\n")
	for _, h := range hints {
		buf.WriteString("  - " + h + "\n")
	}
	return buf.String()
}

func pingFailureHints(server string, err error) []string {
	if err == nil {
		return nil
	}

	var hints []string
	seen := make(map[string]struct{})
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		hints = append(hints, s)
	}

	msg := err.Error()

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		add("DNS lookup failed — check the hostname in --server or run 'remote-agent config'")
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) || strings.Contains(msg, "connection refused") {
			add("connection refused — is ai-critic-server running? is the --server URL and port correct?")
		}
		if errors.Is(opErr.Err, syscall.ETIMEDOUT) || strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "timeout") {
			add("timed out — check network, VPN, firewall, or whether the server is reachable from this machine")
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		for _, h := range pingFailureHints(server, urlErr.Err) {
			add(h)
		}
	}

	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) || strings.Contains(msg, "x509") || strings.Contains(msg, "certificate") {
		add("TLS certificate problem — verify https:// URL and any corporate TLS interception / proxy settings")
	}

	if strings.Contains(msg, "unexpected status") {
		add("server responded but /ping failed — you may be hitting the wrong host or a reverse proxy without /ping")
	}
	if strings.Contains(msg, "unexpected response") {
		add("response was not 'pong' — endpoint may not be ai-critic-server")
	}

	if len(hints) == 0 {
		add(fmt.Sprintf("verify the server URL, run 'remote-agent config', or try: curl -v %s/ping", server))
	}
	return hints
}