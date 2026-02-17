package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/run"
	"github.com/xhd2015/lifelog-private/ai-critic/server"
	"github.com/xhd2015/lifelog-private/ai-critic/server/auth"
)

//go:embed ai-critic-react/dist
var distFS embed.FS

//go:embed ai-critic-react/template.html
var templateHTML string

func main() {
	server.Init(distFS, templateHTML)

	// Check for --quick-test flag before running and set it in server
	// This needs to be set BEFORE run.Run() since RegisterAPI checks quickTestMode
	for _, arg := range os.Args[1:] {
		if arg == "--quick-test" {
			server.SetQuickTestMode(true)
			auth.SetQuickTestMode(true)
		}
		if arg == "--keep" {
			server.SetQuickTestKeep(true)
		}
		if arg == "--frontend-port" {
			// Next arg should be the port
		}
	}

	// Set frontend port (default 18432 for quick-test)
	frontendPort := 18432
	for i, arg := range os.Args[1:] {
		if arg == "--frontend-port" && i+1 < len(os.Args[1:]) {
			fmt.Sscanf(os.Args[i+2], "%d", &frontendPort)
		}
	}
	server.SetFrontendPort(frontendPort)

	err := run.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
