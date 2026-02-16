package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const help = `
Usage: go run ./script/run quick-test [options]

Options:
  -h, --help   Show this help message
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(help)
		return nil
	}

	quickTestPort := lib.QuickTestPort

	// Kill previous process on the port
	fmt.Printf("Checking for existing server on port %d...\n", quickTestPort)
	prevPid, err := lib.GetPidOnPort(quickTestPort)
	if err == nil && prevPid != 0 {
		fmt.Printf("Killing previous server (PID: %d)...\n", prevPid)
		err := syscall.Kill(prevPid, syscall.SIGKILL)
		if err != nil {
			fmt.Printf("Warning: failed to kill previous process: %v\n", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Build the Go server with quick-test flag
	fmt.Println("Building Go server...")
	err = buildServer()
	if err != nil {
		return fmt.Errorf("failed to build Go server: %v", err)
	}

	// Start the server in quick-test mode from home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	fmt.Printf("Starting server in quick-test mode on port %d...\n", quickTestPort)
	fmt.Printf("Running from: %s\n", homeDir)
	serverCmd := exec.Command("/tmp/ai-critic-quick", "--quick-test")
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Stdin = os.Stdin
	serverCmd.Dir = homeDir
	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Go server: %v", err)
	}

	fmt.Printf("Server started with PID: %d\n", serverCmd.Process.Pid)
	fmt.Println("Server will exit after 1 minute of inactivity.")
	fmt.Println("Press Ctrl+C to stop manually.")

	// Wait for the server process
	err = serverCmd.Wait()
	if err != nil {
		return fmt.Errorf("server exited with error: %v", err)
	}
	return nil
}

func buildServer() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}
	projectDir := filepath.Join(homeDir, "mobile-coding-connector")

	cmd := exec.Command("go", "build", "-o", "/tmp/ai-critic-quick", "./")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = projectDir
	return cmd.Run()
}
