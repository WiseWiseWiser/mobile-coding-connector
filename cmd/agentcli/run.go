package agentcli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	agentskill "github.com/xhd2015/ai-critic/cmd/agentcli/skill"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/less-gen/flags"
)

// Run executes the agent CLI with the given profile and arguments.
func Run(profile Profile, args []string) error {
	active = profile
	help := topLevelHelp(profile)

	var server string
	var token string
	var port int
	tokenSpecified := hasGlobalFlag(args, "--token")

	parser := flags.
		String("--server", &server).
		String("--token", &token)
	if profile.SupportsPortFlag {
		parser = parser.Int("--port", &port)
	}
	args, err := parser.
		HelpFunc("-h,--help", func() {
			fmt.Print(strings.TrimRight(help, "\n"))
		}).
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

	if len(args) == 0 {
		fmt.Print(strings.TrimRight(help, "\n"))
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	resolve := func() (*client.Client, error) {
		return resolveClient(server, port, token, tokenSpecified)
	}

	switch cmd {
	case "config":
		return runConfig(rest)
	case "ping":
		return runPing(resolve, rest)
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
	case "git":
		return runGit(resolve, rest)
	case "proxy":
		return runProxy(resolve, rest)
	case "project":
		return runProject(resolve, rest)
	case "settings":
		return runSettings(resolve, rest)
	case "service":
		return runService(resolve, rest)
	case "server":
		return runServer(resolve, rest)
	case "auth":
		return runAuth(resolve, rest)
	case "agent":
		return runAgent(resolve, rest)
	case "skill":
		return agentskill.Handle(rest)
	case "openclaw":
		return runOpenClaw(resolve, rest)
	case "ws-proxy":
		return runWSProxy(resolve, rest)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func resolveClient(server string, port int, token string, tokenSpecified bool) (*client.Client, error) {
	if port > 0 {
		server = fmt.Sprintf("http://localhost:%d", port)
	}

	cfg, _ := loadConfig()

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
		case "--server", "--token", "--port":
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