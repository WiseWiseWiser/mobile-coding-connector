package run

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/lifelog-private/ai-critic/server"
	"github.com/xhd2015/lifelog-private/ai-critic/server/auth"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"

	"github.com/xhd2015/kool/pkgs/web"
	"github.com/xhd2015/less-gen/flags"
)

var help = fmt.Sprintf(`
Usage: ai-critic [options]
       ai-critic keep-alive-script [options]  Output a shell script that auto-restarts the server

Options:
  --dev                   Run in development mode
  --dir DIR               Set the initial directory for code review (defaults to current working directory)
  --port PORT             Port to listen on (defaults to auto-find starting from %d)
  --config-file FILE      Path to configuration file (JSON)
  --credentials-file FILE Path to credentials file (defaults to ".server-credentials")
  --enc-key-file FILE     Path to encryption key file (defaults to ".ai-critic-enc-key")
  --domains-file FILE     Path to domains JSON file (defaults to ".server-domains.json")
  --rules-dir DIR         Directory containing REVIEW_RULES.md (defaults to "rules")
  --component             Serve a specific component
  -h, --help              Show this help message
`, lib.DefaultServerPort)

func Run(args []string) error {
	// Handle subcommands before flag parsing
	if len(args) > 0 {
		switch args[0] {
		case "keep-alive-script":
			return runKeepAliveScript(args[1:])
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
		// Set the AI config in the server
		server.SetAIConfig(cfg)
	}

	// Set credentials file (defaults to ".server-credentials")
	if credentialsFileFlag != "" {
		auth.SetCredentialsFile(credentialsFileFlag)
	}

	// Set encryption key file (defaults to ".ai-critic-enc-key")
	if encKeyFileFlag != "" {
		encrypt.SetKeyFile(encKeyFileFlag)
	}

	// Set domains file (defaults to ".server-domains.json")
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
	var port int
	if portFlag > 0 {
		port = portFlag
		// Check if port is already in use
		if isPortInUse(port) {
			pid := findPortPID(port)
			if pid != "" {
				return fmt.Errorf("port %d is already in use by process %s", port, pid)
			}
			return fmt.Errorf("port %d is already in use", port)
		}
	} else {
		// Auto-find available port starting from DefaultServerPort
		port, err = web.FindAvailablePort(lib.DefaultServerPort, 100)
		if err != nil {
			return err
		}
	}

	// Set server port for domains tunnel management
	domains.SetServerPort(port)

	// Auto-start Cloudflare tunnels for configured domains
	domains.AutoStartTunnels()

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

// findPortPID attempts to find the PID of the process listening on the given port.
func findPortPID(port int) string {
	var cmd *exec.Cmd
	portStr := fmt.Sprintf("%d", port)

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", portStr))
	case "linux":
		// Try lsof first, fall back to ss
		cmd = exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", portStr))
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		// lsof not available on Linux, try ss
		if runtime.GOOS == "linux" {
			cmd = exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%s", portStr))
			out, err = cmd.Output()
			if err != nil {
				return ""
			}
			// Parse ss output for pid
			for _, line := range strings.Split(string(out), "\n") {
				if idx := strings.Index(line, "pid="); idx >= 0 {
					rest := line[idx+4:]
					if end := strings.IndexAny(rest, ",) \t\n"); end > 0 {
						return rest[:end]
					}
				}
			}
			return ""
		}
		return ""
	}

	pid := strings.TrimSpace(string(out))
	// lsof may return multiple PIDs (one per line), take the first
	if idx := strings.IndexByte(pid, '\n'); idx > 0 {
		pid = pid[:idx]
	}
	return pid
}
