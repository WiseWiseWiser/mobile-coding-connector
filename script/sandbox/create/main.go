package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/sandbox/create

Creates a baremetal Debian container and drops you into a shell.
The container is kept running after exit (use "podman rm -f ai-critic-sandbox" to remove).

Steps:
  1. Check podman is installed and machine is running
  2. Create and start a Debian container
  3. Exec into the container with an interactive shell

Options:
  -h, --help    Show this help message
`

func main() {
	_, err := flags.Help("-h,--help", help).Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Step 0: Ensure podman is available and the machine is running
	if err := lib.EnsurePodman(); err != nil {
		return err
	}

	// Step 1: Check if container already exists
	fmt.Println("\n=== Creating container ===")

	inspectCmd := exec.Command("podman", "inspect", "--format", "{{.State.Status}}", lib.ContainerName)
	var inspectBuf bytes.Buffer
	inspectCmd.Stdout = &inspectBuf
	inspectCmd.Stderr = &inspectBuf

	if err := inspectCmd.Run(); err == nil {
		status := strings.TrimSpace(inspectBuf.String())
		if status == "running" {
			fmt.Printf("Container %q is already running. Attaching...\n", lib.ContainerName)
			return execShell()
		}
		// Container exists but is stopped — start it
		fmt.Printf("Container %q exists (status: %s). Starting...\n", lib.ContainerName, status)
		if err := lib.RunVerbose("podman", "start", lib.ContainerName); err != nil {
			return fmt.Errorf("failed to start container: %v", err)
		}
		return execShell()
	}

	// Container doesn't exist — create it
	fmt.Printf("Creating container %q from %s...\n", lib.ContainerName, lib.ContainerImage)
	createArgs := []string{
		"run", "-d",
		"--name", lib.ContainerName,
		lib.ContainerImage,
		"sleep", "infinity",
	}
	if err := lib.RunVerbose("podman", createArgs...); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	return execShell()
}

// execShell opens an interactive shell inside the container.
func execShell() error {
	fmt.Printf("\nDropping into shell in container %q...\n", lib.ContainerName)
	fmt.Println("(Type 'exit' to leave the container. It will keep running.)")

	shellCmd := exec.Command("podman", "exec", "-it", lib.ContainerName, "/bin/bash")
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	return shellCmd.Run()
}
