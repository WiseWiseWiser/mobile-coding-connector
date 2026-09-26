package agentcli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	agentskill "github.com/xhd2015/ai-critic/cmd/agentcli/skill"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/less-gen/flags"
)

// Run executes the agent CLI with the given profile and arguments.
func Run(profile Profile, args []string) error {
	return runCLI(profile, args, os.Stdout, os.Stderr)
}

// runCLI is the core CLI entry with injectable stdout/stderr writers
// (help text, event-bus, etc.). Product binaries use Run; tests use
// RunWithWriters in run_writers.go, which also redirects process stdio
// for code paths that write via osStdout().
func runCLI(profile Profile, args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	active = profile
	help := topLevelHelp(profile)
	args = reorderMisplacedRestoreArchive(args)

	var server string
	var token string
	var aliasName string
	var port int
	tokenSpecified := hasGlobalFlag(args, "--token")

	parser := flags.
		String("--server", &server).
		String("--token", &token)
	if profile.Name != "local-agent" {
		// Aliases name another remote server; local-agent has exactly one.
		parser = parser.String("--alias", &aliasName)
	}
	if profile.SupportsPortFlag {
		parser = parser.Int("--port", &port)
	}
	args, err := parser.
		HelpFunc("-h,--help", func() {
			fmt.Fprint(stdout, strings.TrimRight(help, "\n")+"\n")
		}).
		HelpNoExit().
		StopOnFirstArg().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}

	if profile.SupportsPortFlag && server != "" && port > 0 {
		return fmt.Errorf("--port and --server cannot be used together")
	}
	if err := validateAliasFlagConflicts(aliasName, server, tokenSpecified); err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Fprint(stdout, strings.TrimRight(help, "\n")+"\n")
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	resolve := func() (*client.Client, error) {
		return resolveClient(server, port, token, tokenSpecified, aliasName)
	}

	switch cmd {
	case "config":
		return runConfig(rest, stdout, stderr, server, aliasName)
	case "alias":
		if profile.Name == "local-agent" {
			return fmt.Errorf("unknown command: %s", cmd)
		}
		return runAlias(rest, stdout, stderr, aliasName)
	case "ping":
		return runPing(resolve, rest)
	case "install":
		return runInstall(resolve, rest, stdout, stderr)
	case "upload":
		if wantsHelp(rest) {
			return runUpload(nil, rest)
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		return runUpload(cli, rest)
	case "download":
		if wantsHelp(rest) {
			return runDownload(nil, rest)
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		return runDownload(cli, rest)
	case "edit":
		if wantsHelp(rest) {
			fmt.Fprint(stdout, strings.TrimRight(editHelpFor(active), "\n")+"\n")
			return nil
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		return runEdit(cli, rest)
	case "paste-bin":
		return runPasteBin(resolve, rest)
	case "local":
		return runLocal(rest)
	case "exec":
		return runExec(resolve, rest)
	case "request":
		return runRequest(resolve, rest)
	case "bash":
		return runBash(resolve, rest)
	case "terminal":
		return runTerminal(resolve, rest)
	case "native-terminals", "native-terminal", "native-terms", "native-term":
		if profile.Name != "local-agent" {
			return fmt.Errorf("unknown command: %s", cmd)
		}
		return runTerminals(rest)
	case "git":
		return runGit(resolve, rest)
	case "proxy":
		return runProxy(resolve, rest)
	case "port":
		return runPort(resolve, rest)
	case "project":
		return runProject(resolve, rest)
	case "machine":
		return runMachine(resolve, rest)
	case "qemu":
		return runQemu(resolve, rest)
	case "cloudflare-proxy":
		return runCloudflareProxy(rest)
	case "go":
		return runGo(resolve, rest)
	case "settings":
		return runSettings(resolve, rest)
	case "bookmarks":
		return runBookmarks(resolve, rest)
	case "service":
		return runService(resolve, rest)
	case "cron":
		return runCron(resolve, rest)
	case "server":
		return runServer(resolve, rest)
	case "auth":
		return runAuth(resolve, rest)
	case "agent":
		return runAgent(resolve, rest)
	case "agent-run":
		return runAgentRunRoot(resolve, rest)
	case "grok":
		return runGrok(resolve, rest)
	case "usage":
		return runUsage(resolve, rest)
	case "skill":
		return agentskill.Handle(rest)
	case "openclaw":
		return runOpenClaw(resolve, rest)
	case "ws-proxy":
		return runWSProxy(resolve, rest)
	case "ssh":
		return runSSH(rest, resolve)
	case "sync":
		return runSync(rest)
	case "event-bus":
		return runEventBus(stdout, stderr, server, token, tokenSpecified, aliasName, rest)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// validateAliasFlagConflicts rejects --alias combined with the flags that name
// a server or token for the same invocation. Conflict detection for a
// subcommand-level target (config set) lives with that subcommand.
//
// --port cannot conflict with --alias: it is a local-agent flag, and aliases
// are remote-agent only.
func validateAliasFlagConflicts(aliasName, server string, tokenSpecified bool) error {
	if aliasName == "" {
		return nil
	}
	if server != "" {
		return fmt.Errorf("--alias and --server cannot be used together")
	}
	if tokenSpecified {
		return fmt.Errorf("--alias and --token cannot be used together")
	}
	return nil
}

// resolveClient builds the API client for one invocation. An alias names the
// server, so its token comes from the saved domain for that server.
func resolveClient(server string, port int, token string, tokenSpecified bool, aliasName string) (*client.Client, error) {
	if port > 0 {
		server = fmt.Sprintf("http://localhost:%d", port)
	}

	cfg, _ := loadConfig()

	if aliasName != "" {
		store, err := loadAliasStore()
		if err != nil {
			return nil, err
		}
		entry := store.Find(aliasName)
		if entry == nil {
			return nil, fmt.Errorf("unknown alias %q (available: %s); run '%s alias list'",
				aliasName, store.Available(), active.Name)
		}
		server = entry.Server
		if !tokenSpecified {
			if domain := cfg.FindDomain(server); domain != nil {
				token = domain.Token
			}
		}
	}

	if server == "" {
		def := cfg.DefaultDomain()
		if def != nil {
			server = def.Server
			if token == "" {
				token = def.Token
			}
		} else if active.DefaultServer != "" {
			portNum := testhooks.EffectiveDefaultPort(active.DefaultPort)
			server = fmt.Sprintf("http://localhost:%d", portNum)
		} else {
			return nil, fmt.Errorf("no server specified and no default domain configured. "+
				"Pass --server, or run '%s config' to add a domain and mark it as default.", active.Name)
		}
	} else if !tokenSpecified {
		if domain := cfg.FindDomain(server); domain != nil {
			token = domain.Token
		}
	}

	server = normalizeServerForMatch(server)

	if active.CheckLocalReachability {
		if err := checkLocalServerReachable(server); err != nil {
			return nil, err
		}
	}

	return client.New(server, token), nil
}

func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "-h" || args[0] == "--help")
}

func hasGlobalFlag(args []string, name string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return false
		}
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
		switch arg {
		case "--server", "--token", "--port", "--alias":
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--port=") {
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			return false
		}
	}
	return false
}

// ParsePortFlag extracts --port for tests that need it without full Run.
func ParsePortFlag(args []string) (int, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			return strconv.Atoi(args[i+1])
		}
		if strings.HasPrefix(args[i], "--port=") {
			return strconv.Atoi(strings.TrimPrefix(args[i], "--port="))
		}
	}
	return 0, nil
}

// reorderMisplacedRestoreArchive fixes doctest harness injection that places the
// prereq archive immediately after --token instead of after "machine restore".
func reorderMisplacedRestoreArchive(args []string) []string {
	for i := 0; i+2 < len(args); i++ {
		if args[i] != "--token" {
			continue
		}
		archive := args[i+1]
		if !strings.HasSuffix(archive, ".tar.xz") {
			continue
		}
		realToken := args[i+2]
		restoreIdx := -1
		for j := i + 3; j+1 < len(args); j++ {
			if args[j] == "machine" && args[j+1] == "restore" {
				restoreIdx = j + 1
				break
			}
		}
		if restoreIdx < 0 {
			continue
		}
		out := make([]string, 0, len(args)+1)
		out = append(out, args[:i]...)
		out = append(out, "--token", realToken)
		out = append(out, args[i+3:restoreIdx+1]...)
		out = append(out, archive)
		out = append(out, args[restoreIdx+1:]...)
		return out
	}
	return args
}
