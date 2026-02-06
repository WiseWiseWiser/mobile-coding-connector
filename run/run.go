package run

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/lifelog-private/ai-critic/server"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"

	"github.com/xhd2015/kool/pkgs/web"
	"github.com/xhd2015/less-gen/flags"
)

var help = fmt.Sprintf(`
Usage: ai-critic [options]

Options:
  --dev              Run in development mode
  --dir DIR          Set the initial directory for code review (defaults to current working directory)
  --port PORT        Port to listen on (defaults to auto-find starting from %d)
  --config-file FILE Path to configuration file (JSON)
  --rules-dir DIR    Directory containing REVIEW_RULES.md (defaults to "rules")
  --component        Serve a specific component
  -h, --help         Show this help message
`, lib.DefaultServerPort)

func Run(args []string) error {
	var devFlag bool
	var component string
	var dirFlag string
	var configFile string
	var rulesDir string
	var portFlag int
	args, err := flags.
		Bool("--dev", &devFlag).
		String("--component", &component).
		String("--dir", &dirFlag).
		Int("--port", &portFlag).
		String("--config-file", &configFile).
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
	} else {
		// Auto-find available port starting from DefaultServerPort
		port, err = web.FindAvailablePort(lib.DefaultServerPort, 100)
		if err != nil {
			return err
		}
	}

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
