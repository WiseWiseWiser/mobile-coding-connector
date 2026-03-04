package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/sandbox/boot [options]

Builds the frontend and Go server as a single Linux binary,
then runs it inside a podman container.

Unlike fresh-setup, the container is reused across runs — user changes
inside the container are preserved between restarts.

Options:
  --arch ARCH   Target architecture: auto, amd64, arm64 (default: auto)
  --recreate-container        Destroy existing container and start fresh (prompts for confirmation)
  --force-recreate-container  Same as --recreate-container but skips confirmation
  -h, --help    Show this help message

Steps:
  1. npm install + npm run build (frontend)
  2. GOOS=linux GOARCH=<arch> go build (server with embedded frontend)
  3. Reuse or create podman container
`

func main() {
	if err := lib.RunSandboxBoot(os.Args[1:], lib.SandboxBootOptions{
		Help: help,
		Sandbox: lib.SandboxOptions{
			ScriptSubDir:  "script/sandbox/boot",
			FreshSetup:    false,
			ContainerPort: lib.QuickTestPort,
			ContainerName: lib.ContainerName,
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
