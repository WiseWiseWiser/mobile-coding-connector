package agentcli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
)

const authHelp = `Usage: remote-agent auth <subcommand> [args...]

Auth utilities for checking server connectivity and token validity.

Subcommands:
  status
      Check authentication status against the configured server.
      Verifies the server is reachable and the token is valid.

  import-local
      Local-agent only: import ~/.ai-critic/server-credentials into CLI config.
`

const authStatusHelp = `Usage: remote-agent auth status

Check authentication status against the configured server.
Uses the default server and token from saved config, or --server/--token flags.

Exit code 0 means authenticated; non-zero means unauthenticated or error.
`

func runAuth(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(authHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		return runAuthStatus(resolve, rest)
	case "import-local":
		return runAuthImportLocal(resolve, rest)
	case "-h", "--help":
		fmt.Print(authHelp)
		return nil
	default:
		return fmt.Errorf("unknown auth subcommand: %s", sub)
	}
}

func runAuthStatus(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(authStatusHelp)
			return nil
		}
		return fmt.Errorf("auth status takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Printf("Server: %s\n", cli.Server)

	result, err := cli.AuthStatus()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	if result.OK {
		fmt.Println("Auth: OK")
		return nil
	}

	if !result.Initialized {
		fmt.Println("Auth: not_initialized (server has no credentials set up)")
		return fmt.Errorf("server not initialized")
	}

	fmt.Println("Auth: unauthorized (check your token)")
	return fmt.Errorf("unauthorized")
}

func runAuthImportLocal(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("auth import-local takes no arguments, got %v", args)
	}
	if active.Name != "local-agent" {
		return fmt.Errorf("auth import-local is local-agent-only and unsupported for %s", active.Name)
	}

	token, credPath, err := readFirstLocalCredentialLine()
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &agentConfig{}
	}

	server := fmt.Sprintf("http://localhost:%d", testhooks.EffectiveDefaultPort(active.DefaultPort))
	upsertDomainToken(cfg, server, token)
	cfg.Default = server
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("save local-agent config: %w", err)
	}

	fmt.Printf("Imported local server credentials from %s for %s.\n", credPath, server)
	fmt.Printf("Default server set in ~/.ai-critic/%s.\n", active.ConfigFile)
	return nil
}

func readFirstLocalCredentialLine() (token string, path string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	path = filepath.Join(home, ".ai-critic", "server-credentials")
	f, err := os.Open(path)
	if err != nil {
		return "", path, fmt.Errorf("read local server credentials from %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, path, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", path, fmt.Errorf("read local server credentials from %s: %w", path, err)
	}
	return "", path, fmt.Errorf("no credential token found in %s", path)
}

func upsertDomainToken(cfg *agentConfig, server, token string) {
	for i := range cfg.Domains {
		if normalizeServerForMatch(cfg.Domains[i].Server) == server {
			cfg.Domains[i].Server = server
			cfg.Domains[i].Token = token
			return
		}
	}
	cfg.Domains = append(cfg.Domains, domainConfig{Server: server, Token: token})
}
