package agentcli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
)

const configSetHelp = `Usage: remote-agent config set [--server URL | --alias NAME] [options]

Add a domain, or update the token of an existing one.

Target (first match wins; conflicting targets are an error):
  --server URL     Server base URL to write (also the global --server)
  --alias NAME     Write the server the alias points at (also the global --alias)

Options:
  --token TOKEN    Bearer token (visible in shell history and ps)
  --token-stdin    Read the token from the first line of stdin
  --clear-token    Remove the token for this server, keeping the domain
  --default        Also make this the default domain; on its own it only
                   switches the default (the domain must already exist)
  --dry-run        Print the plan; write nothing
  -h, --help       Show this help message

Tokens are stored per server, so any server — default or not — can be selected
with --server URL, 'config set --default', or an alias (--alias NAME).

Examples:
  printf '%s\n' "$XDEV_TOKEN" | remote-agent config set --server https://agent-xdev.example.com --token-stdin
  xdev-agent config set --token-stdin
  remote-agent --alias xdev config set --default
`

// configTargetSpec is one flag that names the server `config set` writes.
type configTargetSpec struct {
	label  string // rendered flag, for conflict messages
	server string // normalized server URL
	alias  string // alias name when the spec came from an alias
}

// runConfigSet writes a token (or the default selection) for the server this
// invocation points at. globalServer/globalAlias are the already-parsed global
// flags, which name the same target as the subcommand-level flags.
func runConfigSet(args []string, stdout, stderr io.Writer, globalServer, globalAlias string) error {
	var subServer, subAlias, token string
	var tokenStdin, clearToken, setDefault, dryRun bool
	args, err := flags.
		String("--server", &subServer).
		String("--alias", &subAlias).
		String("--token", &token).
		Bool("--token-stdin", &tokenStdin).
		Bool("--clear-token", &clearToken).
		Bool("--default", &setDefault).
		Bool("--dry-run", &dryRun).
		HelpFunc("-h,--help", func() { io.WriteString(stdout, configSetHelp) }).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("config set takes no arguments, got %v; see '%s config set --help'", args, active.Name)
	}

	switch {
	case token != "" && clearToken:
		return fmt.Errorf("--token and --clear-token cannot be used together")
	case token != "" && tokenStdin:
		return fmt.Errorf("--token and --token-stdin cannot be used together")
	case tokenStdin && clearToken:
		return fmt.Errorf("--token-stdin and --clear-token cannot be used together")
	case token == "" && !tokenStdin && !clearToken && !setDefault:
		return fmt.Errorf("config set requires --token, --token-stdin, --clear-token, or --default; see '%s config set --help'", active.Name)
	}
	tokenMode := token != "" || tokenStdin || clearToken

	target, err := resolveConfigSetTarget(subServer, subAlias, globalServer, globalAlias)
	if err != nil {
		return err
	}
	server := target.server

	tokenValue := strings.TrimSpace(token)
	if tokenStdin {
		v, err := readTokenStdin(os.Stdin)
		if err != nil {
			return err
		}
		tokenValue = v
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &agentConfig{}
	}
	store, err := loadAliasStore()
	if err != nil {
		return err
	}
	existing := cfg.FindDomain(server) != nil
	aliases := store.FindByServer(server)

	if !existing && !tokenMode {
		return fmt.Errorf("no domain saved for %s; pass --token or --token-stdin to add it", server)
	}

	if token != "" {
		fmt.Fprintf(stderr, "warning: --token is visible in shell history and ps; prefer --token-stdin\n")
	}
	if clearToken && len(aliases) > 0 {
		fmt.Fprintf(stderr, "warning: %s\n", aliasUsageWarning(len(aliases), aliases))
	}

	if dryRun {
		if tokenMode {
			if !existing {
				fmt.Fprintf(stdout, "[dry-run] would add domain %s\n", server)
			}
			switch {
			case clearToken:
				fmt.Fprintf(stdout, "[dry-run] would clear token for %s\n", server)
			case target.alias != "":
				fmt.Fprintf(stdout, "[dry-run] would set token for %s (alias %s)\n", server, target.alias)
			default:
				fmt.Fprintf(stdout, "[dry-run] would set token for %s\n", server)
			}
		}
		if setDefault {
			fmt.Fprintf(stdout, "[dry-run] would set default domain to %s\n", server)
		}
		return nil
	}

	if tokenMode {
		upsertDomainToken(cfg, server, tokenValue)
	}
	if setDefault {
		cfg.Default = server
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}

	if tokenMode {
		if !existing {
			fmt.Fprintf(stdout, "domain added: %s\n", server)
		}
		annotation := aliasUsageAnnotation(aliases)
		if clearToken {
			fmt.Fprintf(stdout, "token cleared for %s%s\n", server, annotation)
		} else {
			fmt.Fprintf(stdout, "token set for %s%s\n", server, annotation)
		}
	}
	if setDefault {
		fmt.Fprintf(stdout, "default domain set to %s\n", server)
	}
	return nil
}

// resolveConfigSetTarget picks the write target from the subcommand flags
// (--server/--alias) and, failing those, the global flags. Any two specs that
// name different servers are an error.
func resolveConfigSetTarget(subServer, subAlias, globalServer, globalAlias string) (configTargetSpec, error) {
	var specs []configTargetSpec

	addServerSpec := func(label, raw string) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		server, err := normalizeTargetServer(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		specs = append(specs, configTargetSpec{label: label, server: server})
		return nil
	}
	addAliasSpec := func(label, raw string) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		store, err := loadAliasStore()
		if err != nil {
			return err
		}
		entry := store.Find(raw)
		if entry == nil {
			return aliasNotFound(raw, store)
		}
		specs = append(specs, configTargetSpec{label: label, server: entry.Server, alias: raw})
		return nil
	}

	if err := addServerSpec("--server", subServer); err != nil {
		return configTargetSpec{}, err
	}
	if err := addAliasSpec("--alias "+strings.TrimSpace(subAlias), subAlias); err != nil {
		return configTargetSpec{}, err
	}
	if err := addServerSpec("--server", globalServer); err != nil {
		return configTargetSpec{}, err
	}
	if err := addAliasSpec("--alias "+strings.TrimSpace(globalAlias), globalAlias); err != nil {
		return configTargetSpec{}, err
	}

	if len(specs) == 0 {
		return configTargetSpec{}, fmt.Errorf(
			"config set needs a target: pass --server URL, --alias NAME, or use the global --alias/--server")
	}
	first := specs[0]
	for _, s := range specs[1:] {
		if s.server == first.server {
			continue
		}
		msg := fmt.Sprintf("conflicting targets: %s → %s, %s → %s", first.label, first.server, s.label, s.server)
		name, other := first.alias, s.server
		if first.alias == "" {
			name, other = s.alias, first.server
		}
		msg += fmt.Sprintf("\nhint: to change which server an alias points at, run '%s alias update %s --server %s'",
			active.Name, name, other)
		return configTargetSpec{}, errors.New(msg)
	}
	return first, nil
}

// normalizeTargetServer trims a target server URL for domain matching.
func normalizeTargetServer(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\n") {
		return "", fmt.Errorf("invalid server %q", raw)
	}
	return normalizeServerForMatch(raw), nil
}

// readTokenStdin reads the first line of r as a bearer token.
func readTokenStdin(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read token from stdin: %w", err)
	}
	line, _, _ := strings.Cut(string(data), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("no token on stdin")
	}
	return line, nil
}

// aliasUsageAnnotation renders " (aliases: a, b)" when aliases target a server.
func aliasUsageAnnotation(aliases []aliasEntry) string {
	if len(aliases) == 0 {
		return ""
	}
	names := make([]string, 0, len(aliases))
	for _, a := range aliases {
		names = append(names, a.Name)
	}
	return fmt.Sprintf(" (aliases: %s)", strings.Join(names, ", "))
}

// aliasUsageWarning renders "2 aliases use this server: a, b".
func aliasUsageWarning(n int, aliases []aliasEntry) string {
	names := make([]string, 0, len(aliases))
	for _, a := range aliases {
		names = append(names, a.Name)
	}
	noun := "aliases use"
	if n == 1 {
		noun = "alias uses"
	}
	return fmt.Sprintf("%d %s this server: %s", n, noun, strings.Join(names, ", "))
}
