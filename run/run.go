package run

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server"
	"github.com/xhd2015/lifelog-private/ai-critic/server/auth"
	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"

	"github.com/xhd2015/less-gen/flags"
)

var help = fmt.Sprintf(`
Usage: ai-critic [options]
       ai-critic keep-alive [options]            Auto-restart server with health checking
       ai-critic keep-alive request <action>     Request action from keep-alive daemon (info, restart)
       ai-critic rebuild --repo-dir DIR [opts]   Rebuild from source and restart
       ai-critic check-port --port PORT          Check if a port is accessible

Options:
  --dev                   Run in development mode
  --dir DIR               Set the initial directory for code review (defaults to current working directory)
  --port PORT             Port to listen on (defaults to auto-find starting from %d)
  --config-file FILE      Path to configuration file (JSON)
  --credentials-file FILE Path to credentials file (defaults to "%s")
  --enc-key-file FILE     Path to encryption key file (defaults to "%s")
  --domains-file FILE     Path to domains JSON file (defaults to "%s")
  --rules-dir DIR         Directory containing REVIEW_RULES.md (defaults to "rules")
  --component             Serve a specific component
  -h, --help              Show this help message

Keep-Alive Options:
  --script                Output shell script instead of running Go code
  --forever               Skip port-in-use check and start keep-alive anyway

Request Actions:
  info                    Get current status from keep-alive daemon
  restart                 Request keep-alive daemon to restart the server
`, config.DefaultServerPort, config.CredentialsFile, config.EncKeyFile, config.DomainsFile)

func Run(args []string) error {
	// Handle subcommands before flag parsing
	if len(args) > 0 {
		switch args[0] {
		case "keep-alive":
			// Check for "keep-alive request" sub-subcommand
			if len(args) > 1 && args[1] == "request" {
				return runKeepAliveRequest(args[2:])
			}
			return runKeepAlive(args[1:])
		case "keep-alive-script":
			// Backward-compatible alias: same as "keep-alive --script"
			return runKeepAlive(append([]string{"--script"}, args[1:]...))
		case "rebuild":
			return runRebuild(args[1:])
		case "rebuild-script":
			// Backward-compatible alias: same as "rebuild --script"
			return runRebuild(append([]string{"--script"}, args[1:]...))
		case "check-port":
			return runCheckPort(args[1:])
		}
	}

	var devFlag bool
	var component string
	var dirFlag string
	var configFile string
	var credentialsFileFlag string
	var encKeyFileFlag string
	var domainsFileFlag string
	var rulesDir string
	var portFlag int
	args, err := flags.
		Bool("--dev", &devFlag).
		String("--component", &component).
		String("--dir", &dirFlag).
		Int("--port", &portFlag).
		String("--config-file", &configFile).
		String("--credentials-file", &credentialsFileFlag).
		String("--enc-key-file", &encKeyFileFlag).
		String("--domains-file", &domainsFileFlag).
		String("--rules-dir", &rulesDir).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra args: %s", strings.Join(args, " "))
	}

	if component == "list" {
		fmt.Println("Available components: App")
		return nil
	}

	// Load config file if specified
	if configFile != "" {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %v", err)
		}
		fmt.Printf("Loaded config from %s\n", configFile)
		// Set the legacy config for non-AI settings
		config.Set(cfg)
		// Set the config file path for saving server settings
		server.SetConfigFilePath(configFile)
	}

	// Load AI configuration (from new file if exists, otherwise from legacy)
	aiCfg, err := config.GetEffectiveAIConfig(config.Get())
	if err != nil {
		return fmt.Errorf("failed to load AI config: %v", err)
	}
	// Set the AI config in the server
	server.SetAIConfigAdapter(aiCfg)

	if credentialsFileFlag != "" {
		auth.SetCredentialsFile(credentialsFileFlag)
	}
	if encKeyFileFlag != "" {
		encrypt.SetKeyFile(encKeyFileFlag)
	}
	if domainsFileFlag != "" {
		domains.SetDomainsFile(domainsFileFlag)
	}

	// Set initial directory (defaults to current working directory)
	initialDir := dirFlag
	if initialDir == "" {
		initialDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
	}
	server.SetInitialDir(initialDir)

	// Set rules directory (defaults to "rules" in current directory)
	if rulesDir != "" {
		server.SetRulesDir(rulesDir)
	}

	// Determine port to use
	port := portFlag
	if port <= 0 {
		port = config.DefaultServerPort
	}
	// Check if port is already in use
	if isPortInUse(port) {
		pid := findPortPID(port)
		if pid != "" {
			return fmt.Errorf("port %d is already in use by process %s", port, pid)
		}
		return fmt.Errorf("port %d is already in use", port)
	}

	// Set server port for domains tunnel management
	domains.SetServerPort(port)

	// Auto-start Cloudflare tunnels for configured domains
	domains.AutoStartTunnels()

	// Start unified tunnel manager global health checks
	// This monitors all mappings (domains + port forwards) and restarts them after 3 failures
	cloudflare.StartGlobalHealthChecks()

	if component != "" {
		var html string
		if !devFlag {
			html, err = server.FormatTemplateHtml(server.FormatOptions{
				Component: component,
			})
			if err != nil {
				return err
			}
		}
		return server.ServeComponent(port, server.ServeOptions{
			Dev: devFlag,
			Static: server.StaticOptions{
				IndexHtml: html,
			},
			OpenBrowserUrl: func(port int, url string) string {
				if devFlag {
					return fmt.Sprintf("%s/?component=%s", url, component)
				}
				return url
			},
		})
	}

	return server.Serve(port, devFlag)
}

// isPortInUse checks if the given port is already in use.
func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findPortPID attempts to find the PID of the process LISTENING on the given port.
func findPortPID(port int) string {
	portStr := fmt.Sprintf("%d", port)

	switch runtime.GOOS {
	case "darwin", "linux":
		// Try lsof with -sTCP:LISTEN to find only the listening socket
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", portStr), "-sTCP:LISTEN")
		out, err := cmd.Output()
		if err == nil {
			pid := strings.TrimSpace(string(out))
			if idx := strings.IndexByte(pid, '\n'); idx > 0 {
				pid = pid[:idx]
			}
			if pid != "" {
				return pid
			}
		}
		// Fallback on Linux: try ss
		if runtime.GOOS == "linux" {
			cmd = exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%s", portStr))
			out, err = cmd.Output()
			if err != nil {
				return ""
			}
			for _, line := range strings.Split(string(out), "\n") {
				if idx := strings.Index(line, "pid="); idx >= 0 {
					rest := line[idx+4:]
					if end := strings.IndexAny(rest, ",) \t\n"); end > 0 {
						return rest[:end]
					}
				}
			}
		}
		return ""
	default:
		return ""
	}
}
