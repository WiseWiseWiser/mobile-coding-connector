package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/sandbox/boot/quick-test [options]

Builds the Go server as a Linux binary (no frontend build),
then runs it in quick-test mode inside a podman container.

The frontend dev server (Vite) runs on the host at port 5173,
and the server inside the container proxies frontend requests to it.

Options:
  --arch ARCH   Target architecture: auto, amd64, arm64 (default: auto)
  --recreate-container        Destroy existing container and start fresh (prompts for confirmation)
  --force-recreate-container  Same as --recreate-container but skips confirmation
  -h, --help    Show this help message
`

func main() {
	containerName := lib.ContainerName
	parsed, err := lib.ParseSandboxCLI(os.Args[1:], help, containerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if parsed == nil {
		return
	}

	if err := lib.RunSandboxQuickTest(lib.SandboxQuickTestOptions{
		ArchFlag:      parsed.ArchFlag,
		ContainerPort: lib.QuickTestPort,
		ContainerName: containerName,
		ScriptSubDir:  "script/sandbox/boot",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
