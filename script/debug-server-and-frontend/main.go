package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const help = `Usage: go run ./script/debug-server-and-frontend [options]

Starts a quick-test server with the latest code and opens a browser debugger.

This script runs quick-test (which manages vite and server) and opens a browser debugger for JS code evaluation.

Options:
  -h, --help      Show this help message
  --port PORT     Port for quick-test server (default: 3580)
  --no-headless   Run browser with visible window
  --no-vite       Pass to quick-test: don't auto-start vite (use built frontend)
`

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var opts lib.QuickTestOptions
	var noHeadless bool

	args, err := flags.
		Int("--port", &opts.Port).
		Bool("--no-headless", &noHeadless).
		Bool("--no-vite", &opts.NoVite).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unknown args: %v", args)
	}

	headless := !noHeadless
	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %v", err)
	}
	opts.ProjectDir = projectRoot

	err = lib.QuickTestPrepare(&opts)
	if err != nil {
		return err
	}

	result, err := lib.QuickTestStart(ctx, &opts)
	if err != nil {
		return err
	}

	fmt.Printf("Waiting for server to be ready on port %d...\n", opts.GetPort())
	if err := waitForPort(ctx, opts.GetPort(), 60*time.Second); err != nil {
		if result.ServerCmd.Process != nil {
			result.ServerCmd.Process.Kill()
		}
		if result.ViteCmd != nil && result.ViteCmd.Process != nil {
			result.ViteCmd.Process.Kill()
		}
		return fmt.Errorf("server failed to start: %v", err)
	}
	fmt.Println("Server is ready!")

	fmt.Println("Starting browser debugger...")
	debugCmd := exec.CommandContext(ctx, "go", "run", "./script/debug-port", fmt.Sprintf("--port=%d", opts.GetPort()))
	if !headless {
		debugCmd.Args = append(debugCmd.Args, "--no-headless")
	}
	debugCmd.Dir = projectRoot
	debugCmd.Stdin = os.Stdin
	debugCmd.Stdout = os.Stdout
	debugCmd.Stderr = os.Stderr

	debugErr := debugCmd.Run()

	if result.ServerCmd.Process != nil {
		fmt.Println("Stopping quick-test server...")
		result.ServerCmd.Process.Signal(syscall.SIGTERM)
		result.ServerCmd.Wait()
	}

	if result.ViteCmd != nil && result.ViteCmd.Process != nil {
		fmt.Println("Stopping Vite dev server...")
		result.ViteCmd.Process.Signal(syscall.SIGTERM)
		result.ViteCmd.Wait()
	}

	if debugErr != nil {
		return fmt.Errorf("debug-port exited with error: %v", debugErr)
	}

	return nil
}

func waitForPort(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if lib.CheckPort(port) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}

func getProjectRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output[:len(output)-1]), nil
}
