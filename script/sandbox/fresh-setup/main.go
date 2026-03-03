package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/sandbox/fresh-setup [options]

Builds the frontend and Go server as a single Linux binary,
then runs it inside a podman container.

Options:
  --arch ARCH   Target architecture: auto, amd64, arm64 (default: auto)
  -h, --help    Show this help message

Steps:
  1. npm install + npm run build (frontend)
  2. GOOS=linux GOARCH=<arch> go build (server with embedded frontend)
  3. podman create + podman cp + podman start
`

func main() {
	var archFlag string
	_, err := flags.
		String("--arch", &archFlag).
		Help("-h,--help", help).
		Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if archFlag == "" {
		archFlag = "auto"
	}

	if err := lib.RunSandbox(lib.SandboxOptions{
		ArchFlag:      archFlag,
		ScriptSubDir:  "script/sandbox/fresh-setup",
		FreshSetup:    true,
		ContainerPort: lib.QuickTestPort,
		ContainerName: lib.ContainerNameFresh,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
