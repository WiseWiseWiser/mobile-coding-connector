package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/run quick-test [options]

Options:
  -h, --help               Show this help message
  --keep                   Keep server running indefinitely (disable auto-shutdown)
  --dev                    Run in development mode (auto-start vite, proxy frontend to 5173)
  --frontend-port PORT     Proxy frontend to PORT (assumes vite/frontend started externally)
  --port PORT              Port to run on (default: 37651)
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var keepFlag bool
	var devFlag bool
	var frontendPortFlag int
	var portFlag int

	args, err := flags.
		Bool("--keep", &keepFlag).
		Bool("--dev", &devFlag).
		Int("--frontend-port", &frontendPortFlag).
		Int("--port", &portFlag).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unknown args: %v", args)
	}

	quickTestPort := lib.QuickTestPort
	if portFlag > 0 {
		quickTestPort = portFlag
	}

	// Kill any existing process on the port
	fmt.Printf("Checking for existing server on port %d...\n", quickTestPort)
	killedPid, err := lib.KillPortPid(quickTestPort)
	if err != nil {
		return err
	}
	if killedPid > 0 {
		fmt.Printf("Killed previous server (PID: %d)\n", killedPid)
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

	serverArgs := []string{"--quick-test"}
	if devFlag {
		serverArgs = append(serverArgs, "--dev")
	}
	if frontendPortFlag > 0 {
		serverArgs = append(serverArgs, "--frontend-port", fmt.Sprintf("%d", frontendPortFlag))
	}
	if keepFlag {
		serverArgs = append(serverArgs, "--keep")
	}
	serverCmd := exec.Command("/tmp/ai-critic-quick", serverArgs...)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Stdin = os.Stdin
	serverCmd.Dir = homeDir
	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Go server: %v", err)
	}

	fmt.Printf("Server started with PID: %d\n", serverCmd.Process.Pid)
	if keepFlag {
		fmt.Println("Server will keep running indefinitely (--keep enabled).")
	} else {
		fmt.Println("Server will exit after 10 minutes of inactivity.")
	}
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
