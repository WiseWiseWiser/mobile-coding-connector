package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
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
  --reset       Destroy existing container and start fresh
  -h, --help    Show this help message

Steps:
  1. npm install + npm run build (frontend)
  2. GOOS=linux GOARCH=<arch> go build (server with embedded frontend)
  3. Reuse or create podman container
`

func main() {
	var archFlag string
	var reset bool
	_, err := flags.
		String("--arch", &archFlag).
		Bool("--reset", &reset).
		Help("-h,--help", help).
		Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if archFlag == "" {
		archFlag = "auto"
	}

	if reset {
		if _, err := lib.InspectContainerStatus(lib.ContainerName); err == nil {
			if isStdinTTY() {
				fmt.Printf("Container %q exists. Destroy and recreate? [y/N] ", lib.ContainerName)
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(answer)) != "y" {
					fmt.Println("Aborted.")
					return
				}
			}
			fmt.Println("Resetting: removing existing container...")
			_ = lib.RunVerbose("podman", "rm", "-f", lib.ContainerName)
		}
	}

	if err := lib.RunSandbox(lib.SandboxOptions{
		ArchFlag:      archFlag,
		ScriptSubDir:  "script/sandbox/boot",
		FreshSetup:    false,
		ContainerPort: lib.QuickTestPort,
		ContainerName: lib.ContainerName,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func isStdinTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
