package agentcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
	"github.com/xhd2015/ai-critic/server/domains"
	"github.com/xhd2015/less-gen/flags"
)

const cfProxyHelpLocal = `Usage: local-agent cloudflare-proxy <command> [OPTIONS]

HTTP/WS reverse-proxy so a remote client can publish hostnames
without running cloudflared. Listens on 127.0.0.1:23790.

Commands:
  status     listen port, domain, auth source, mapping count
  start      start now (rotates generatedToken when token is empty)
  stop       stop listening
  enable     auto-start on local-agent keep-alive (use --now to also start)
  disable    clear auto-start (use --now to also stop)
  add        publish a hostname and serve until Ctrl-C
  list       mappings for this token
  delete     remove a mapping (disconnects a blocking add)

Options:
  --hostname HOST
  --echo                 handle requests in-process (debug)
  --forward URL          reverse-proxy to URL (e.g. http://127.0.0.1:23712)
  --proxy-url URL        proxy base (default: saved cloudflare.json or http://127.0.0.1:23790)
  --token TOKEN          CRUD token
  --id ID                delete
  --now                  with enable/disable: also start/stop
  --foreground           start: run in this terminal (no keep-alive daemon)
  --port N               listen port (default 23790)
  --domain HOST          persist proxy domain
  -h, --help

Config: ~/.ai-critic/cloudflare-proxy/config.json
Client: ~/.ai-critic/cloudflare.json (proxy_url, token)

Run 'local-agent cloudflare-proxy <command> --help' for command-specific options.
`

const cfProxyHelpRemote = `Usage: remote-agent cloudflare-proxy <command> [OPTIONS]

Client for an ext.dev cloudflare-proxy. Does not run cloudflared here.

Commands:
  add       publish a hostname and serve until Ctrl-C
  list      mappings for this token
  delete    remove a mapping (disconnects a blocking add)

Options:
  --hostname HOST
  --echo                 handle requests in-process (debug)
  --forward URL          reverse-proxy to URL (e.g. http://127.0.0.1:23712)
  --proxy-url URL        proxy base (default: saved cloudflare.json)
  --token TOKEN          CRUD token
  --id ID                delete
  -h, --help

Client config: ~/.ai-critic/cloudflare.json (proxy_url, token)

Run 'remote-agent cloudflare-proxy <command> --help' for command-specific options.
`

const cfProxyAddHelp = `Usage: %s cloudflare-proxy add --hostname HOST (--echo | --forward URL) [OPTIONS]

Create a mapping, dial the proxy WebSocket pool, and serve until Ctrl-C.
Ctrl-C deletes the mapping.

Options:
  --hostname HOST
  --echo                 in-process 200 + request dump
  --forward URL          reverse-proxy to URL
  --proxy-url URL
  --token TOKEN
  -h, --help
`

const cfProxyListHelp = `Usage: %s cloudflare-proxy list [OPTIONS]

List mappings for this token.

Options:
  --proxy-url URL
  --token TOKEN
  -h, --help
`

const cfProxyDeleteHelp = `Usage: %s cloudflare-proxy delete [--id ID | --hostname HOST] [OPTIONS]

Remove a mapping. A blocking add serving that mapping exits.

Options:
  --id ID
  --hostname HOST
  --proxy-url URL
  --token TOKEN
  -h, --help
`

const cfProxyStatusHelp = `Usage: local-agent cloudflare-proxy status

Show listen port, domain, auth source, and mapping count.
`

const cfProxyStartHelp = `Usage: local-agent cloudflare-proxy start [OPTIONS]

Start the proxy listener now. Rotates generatedToken when token is empty.

Options:
  --foreground    run in this terminal
  --port N
  --domain HOST
  --token TOKEN   persist configured token
  -h, --help
`

const cfProxyStopHelp = `Usage: local-agent cloudflare-proxy stop

Stop the proxy listener. Overlay/config is kept.
`

const cfProxyEnableHelp = `Usage: local-agent cloudflare-proxy enable [--now]

Persist auto-start on local-agent keep-alive.

Options:
  --now    also start now
  -h, --help
`

const cfProxyDisableHelp = `Usage: local-agent cloudflare-proxy disable [--now]

Clear auto-start.

Options:
  --now    also stop now
  -h, --help
`

type cfProxyFlags struct {
	Hostname   string
	Echo       bool
	Forward    string
	ProxyURL   string
	Token      string
	ID         string
	Now        bool
	Foreground bool
	Port       int
	Domain     string
}

func runCloudflareProxy(args []string) error {
	local := active.Name == "local-agent"
	top := cfProxyHelpRemote
	if local {
		top = cfProxyHelpLocal
	}
	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Print(strings.TrimPrefix(top, "\n"))
		return nil
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "status", "start", "stop", "enable", "disable":
		if !local {
			return fmt.Errorf("start/stop/enable/disable are local-agent-only")
		}
		return runCFProxyDaemon(cmd, rest)
	case "add":
		return runCFProxyAdd(rest)
	case "list":
		return runCFProxyList(rest)
	case "delete":
		return runCFProxyDelete(rest)
	default:
		return fmt.Errorf("unknown cloudflare-proxy command %q", cmd)
	}
}

func parseCFProxyFlags(args []string, help string) (cfProxyFlags, []string, error) {
	var f cfProxyFlags
	rest, err := flags.
		String("--hostname", &f.Hostname).
		Bool("--echo", &f.Echo).
		String("--forward", &f.Forward).
		String("--proxy-url", &f.ProxyURL).
		String("--token", &f.Token).
		String("--id", &f.ID).
		Bool("--now", &f.Now).
		Bool("--foreground", &f.Foreground).
		Int("--port", &f.Port).
		String("--domain", &f.Domain).
		Help("-h,--help", help).
		HelpNoExit().
		Parse(args)
	if err != nil {
		return f, nil, err
	}
	return f, rest, nil
}

func runCFProxyDaemon(action string, args []string) error {
	help := cfProxyStatusHelp
	switch action {
	case "start":
		help = cfProxyStartHelp
	case "stop":
		help = cfProxyStopHelp
	case "enable":
		help = cfProxyEnableHelp
	case "disable":
		help = cfProxyDisableHelp
	}
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Print(strings.TrimPrefix(help, "\n"))
		return nil
	}
	f, _, err := parseCFProxyFlags(args, help)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	base, err := proxyBaseDir()
	if err != nil {
		return err
	}
	if action == "start" && f.Foreground {
		return runCFProxyForeground(base, f)
	}
	req := cloudflareproxy.OpRequest{Op: action, Now: f.Now, Port: f.Port, Domain: f.Domain, Token: f.Token}
	resp, err := cloudflareproxy.ControlClient(cloudflareproxy.SocketPath(base), req)
	if err != nil {
		return err
	}
	return printCFProxyResult(resp)
}

func runCFProxyForeground(base string, f cfProxyFlags) error {
	cfg, err := cloudflareproxy.LoadConfig(cloudflareproxy.ConfigPath(base))
	if err != nil {
		return err
	}
	if f.Port > 0 {
		cfg.Port = f.Port
	}
	if f.Domain != "" {
		cfg.Domain = f.Domain
	}
	if f.Token != "" {
		cfg.Token = f.Token
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = cloudflareproxy.RunForeground(ctx, base, cfg, proxyBindDomain, func(warns []string, live cloudflareproxy.Config) {
		for _, w := range warns {
			fmt.Fprintln(os.Stderr, w)
		}
		port := live.Port
		if port <= 0 {
			port = cloudflareproxy.DefaultPort
		}
		kv := map[string]string{
			"listen":    fmt.Sprintf("127.0.0.1:%d", port),
			"status":    "running",
			"domain":    live.Domain,
			"auth":      liveAuth(live),
			"autoStart": fmt.Sprintf("%v", live.AutoStart),
			"mappings":  "0",
			"connected": "0",
		}
		if strings.TrimSpace(live.Token) == "" {
			kv["auth_rotated"] = "true"
		}
		printCFProxyStatus(kv)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func liveAuth(cfg cloudflareproxy.Config) string {
	if strings.TrimSpace(cfg.Token) != "" {
		return "token"
	}
	if strings.TrimSpace(cfg.GeneratedToken) != "" {
		return "generatedToken"
	}
	return "none"
}

func runCFProxyAdd(args []string) error {
	help := fmt.Sprintf(cfProxyAddHelp, active.Name)
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Print(strings.TrimPrefix(help, "\n"))
		return nil
	}
	f, _, err := parseCFProxyFlags(args, help)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if !f.Echo && strings.TrimSpace(f.Forward) == "" {
		return fmt.Errorf("need --echo or --forward URL")
	}
	if f.Echo && strings.TrimSpace(f.Forward) != "" {
		return fmt.Errorf("--echo and --forward are mutually exclusive")
	}
	cli, err := newProxyAPIClient(f)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = cli.Publish(ctx, cloudflareproxy.PublishOpts{
		Hostname: f.Hostname,
		Echo:     f.Echo,
		Forward:  f.Forward,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	})
	if err != nil && errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func runCFProxyList(args []string) error {
	help := fmt.Sprintf(cfProxyListHelp, active.Name)
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Print(strings.TrimPrefix(help, "\n"))
		return nil
	}
	f, _, err := parseCFProxyFlags(args, help)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	cli, err := newProxyAPIClient(f)
	if err != nil {
		return err
	}
	list, err := cli.ListMappings()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("0 mappings")
		return nil
	}
	fmt.Printf("  %-10s  %-24s  %4s  %s\n", "ID", "HOSTNAME", "WS", "URL")
	for _, m := range list {
		fmt.Printf("  %-10s  %-24s  %4d  %s\n", m.ID, m.Hostname, m.WS, m.URL)
	}
	fmt.Printf("\n%d mapping", len(list))
	if len(list) != 1 {
		fmt.Print("s")
	}
	fmt.Println()
	return nil
}

func runCFProxyDelete(args []string) error {
	help := fmt.Sprintf(cfProxyDeleteHelp, active.Name)
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Print(strings.TrimPrefix(help, "\n"))
		return nil
	}
	f, rest, err := parseCFProxyFlags(args, help)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if f.ID == "" && len(rest) > 0 {
		f.ID = rest[0]
	}
	if f.ID == "" && f.Hostname == "" {
		return fmt.Errorf("need --id or --hostname")
	}
	cli, err := newProxyAPIClient(f)
	if err != nil {
		return err
	}
	deleted, err := cli.DeleteMapping(f.ID, f.Hostname)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("mapping %q not found", firstNonEmpty(f.ID, f.Hostname))
		}
		return err
	}
	fmt.Printf("deleted %s\n", deleted)
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func newProxyAPIClient(f cfProxyFlags) (*cloudflareproxy.APIClient, error) {
	proxyURL := strings.TrimSpace(f.ProxyURL)
	token := strings.TrimSpace(f.Token)
	if proxyURL == "" || token == "" {
		cf, err := loadClientCloudflareFile()
		if err != nil {
			return nil, err
		}
		if proxyURL == "" {
			proxyURL = strings.TrimSpace(cf.ProxyURL)
		}
		if token == "" {
			token = strings.TrimSpace(cf.Token)
		}
	}
	if proxyURL == "" && active.Name == "local-agent" {
		proxyURL = fmt.Sprintf("http://127.0.0.1:%d", cloudflareproxy.DefaultPort)
	}
	if proxyURL == "" {
		return nil, fmt.Errorf("proxy_url is required (--proxy-url or ~/.ai-critic/cloudflare.json)")
	}
	if token == "" && active.Name == "local-agent" {
		base, err := proxyBaseDir()
		if err == nil {
			if cfg, err := cloudflareproxy.LoadConfig(cloudflareproxy.ConfigPath(base)); err == nil {
				token = cfg.AuthToken()
			}
		}
	}
	if token == "" {
		return nil, fmt.Errorf("token is required (--token or ~/.ai-critic/cloudflare.json)")
	}
	return &cloudflareproxy.APIClient{BaseURL: proxyURL, Token: token}, nil
}

func loadClientCloudflareFile() (cloudflareproxy.ClientFile, error) {
	home, err := testhooks.UserHomeDir()
	if err != nil {
		return cloudflareproxy.ClientFile{}, err
	}
	return cloudflareproxy.LoadClientFile(filepath.Join(home, ".ai-critic", "cloudflare.json"))
}

func proxyBaseDir() (string, error) {
	home, err := testhooks.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", cloudflareproxy.DirName), nil
}

func printCFProxyResult(resp cloudflareproxy.OpResponse) error {
	if resp.Output != "" {
		for _, line := range strings.Split(strings.TrimRight(resp.Output, "\n"), "\n") {
			if strings.HasPrefix(line, "warning:") {
				fmt.Fprintln(os.Stderr, line)
				continue
			}
			fmt.Println(line)
		}
	}
	if !resp.OK {
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		return fmt.Errorf("action failed")
	}
	printCFProxyStatus(resp.KV)
	return nil
}

func printCFProxyStatus(kv map[string]string) {
	if kv == nil {
		return
	}
	listen := kv["listen"]
	status := kv["status"]
	if listen != "" {
		if status != "" {
			fmt.Printf("listen:     %s  %s\n", listen, status)
		} else {
			fmt.Printf("listen:     %s\n", listen)
		}
	}
	domain := kv["domain"]
	if domain == "" {
		fmt.Println("domain:     (none)")
	} else {
		bound := "not bound"
		if kv["domain_bound"] == "yes" {
			bound = "bound"
		}
		fmt.Printf("domain:     %s  %s\n", domain, bound)
	}
	auth := kv["auth"]
	if kv["auth_rotated"] == "true" {
		auth = "generatedToken (rotated)"
	}
	if auth != "" {
		fmt.Printf("auth:       %s\n", auth)
	}
	if v := kv["autoStart"]; v != "" {
		fmt.Printf("autoStart:  %s\n", v)
	}
	if m := kv["mappings"]; m != "" {
		c := kv["connected"]
		if c == "" {
			c = "0"
		}
		fmt.Printf("mappings:   %s  (%s connected)\n", m, c)
	}
}

func proxyBindDomain(domain string, port int) error {
	_, err := domains.StartHostDomainTunnel(domain, port, func(msg string) {
		fmt.Fprintf(os.Stderr, "bind %s: %s\n", domain, msg)
	})
	return err
}
