package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const defaultQuickTestPort = lib.QuickTestPort
const viteDevPort = lib.ViteDevPort

const help = `Usage: go run ./script/debug-server-and-frontend [options]

Starts a quick-test server with the latest code and opens a browser debugger.

This script:
  - Kills any existing server on the port
  - Starts the Vite dev server for frontend hot reload
  - Builds and starts a fresh server with latest code (proxies to vite)
  - Opens a browser debugger for testing

Options:
  -h, --help      Show this help message
  --port PORT     Port for quick-test server (default: 37651)
  --no-headless   Run browser with visible window
  --no-vite       Skip starting vite dev server (use built frontend)
`

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var port int
	var noHeadless bool
	var noVite bool

	args, err := flags.
		Int("--port", &port).
		Bool("--no-headless", &noHeadless).
		Bool("--no-vite", &noVite).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unknown args: %v", args)
	}

	headless := !noHeadless

	if port == 0 {
		port = defaultQuickTestPort
	}

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

	// Kill any existing process on the port
	fmt.Printf("Checking for existing server on port %d...\n", port)
	killedPid, err := lib.KillPortPid(port)
	if err != nil {
		return err
	}
	if killedPid > 0 {
		fmt.Printf("Killed previous server (PID: %d)\n", killedPid)
	}

	var viteCmd *exec.Cmd
	if !noVite {
		// Start vite dev server
		fmt.Println("Starting Vite dev server...")
		viteCmd = exec.CommandContext(ctx, "npm", "run", "dev")
		viteCmd.Dir = projectRoot + "/ai-critic-react"
		viteCmd.Stdout = os.Stdout
		viteCmd.Stderr = os.Stderr

		if err := viteCmd.Start(); err != nil {
			return fmt.Errorf("failed to start vite dev server: %v", err)
		}

		// Wait for vite to be ready
		fmt.Printf("Waiting for Vite dev server on port %d...\n", viteDevPort)
		if err := waitForHTTP(ctx, fmt.Sprintf("http://localhost:%d", viteDevPort), 30*time.Second); err != nil {
			if viteCmd.Process != nil {
				viteCmd.Process.Kill()
			}
			return fmt.Errorf("vite dev server failed to start: %v", err)
		}
		fmt.Println("Vite dev server is ready!")
	}

	// Start quick-test server with --frontend-port to proxy to vite (vite started externally)
	fmt.Printf("Starting quick-test server on port %d...\n", port)
	quickTestArgs := []string{"run", "./script/run/quick-test", fmt.Sprintf("--port=%d", port), "--frontend-port", "5173"}
	quickTestCmd := exec.CommandContext(ctx, "go", quickTestArgs...)
	quickTestCmd.Dir = projectRoot
	quickTestCmd.Stdout = os.Stdout
	quickTestCmd.Stderr = os.Stderr

	if err := quickTestCmd.Start(); err != nil {
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		return fmt.Errorf("failed to start quick-test server: %v", err)
	}

	fmt.Printf("Waiting for server to be ready on port %d...\n", port)
	if err := waitForPort(ctx, port, 30*time.Second); err != nil {
		quickTestCmd.Process.Kill()
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		return fmt.Errorf("server failed to start: %v", err)
	}
	fmt.Println("Server is ready!")

	// Start debug-port
	fmt.Println("Starting browser debugger...")
	debugCmd := exec.CommandContext(ctx, "go", "run", "./script/debug-port", fmt.Sprintf("--port=%d", port))
	if !headless {
		debugCmd.Args = append(debugCmd.Args, "--no-headless")
	}
	debugCmd.Dir = projectRoot
	debugCmd.Stdin = os.Stdin
	debugCmd.Stdout = os.Stdout
	debugCmd.Stderr = os.Stderr

	debugErr := debugCmd.Run()

	// Clean up quick-test server
	if quickTestCmd.Process != nil {
		fmt.Println("Stopping quick-test server...")
		quickTestCmd.Process.Signal(syscall.SIGTERM)
		quickTestCmd.Wait()
	}

	// Clean up vite server
	if viteCmd != nil && viteCmd.Process != nil {
		fmt.Println("Stopping Vite dev server...")
		viteCmd.Process.Signal(syscall.SIGTERM)
		viteCmd.Wait()
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

func waitForHTTP(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for HTTP at %s", url)
}

func getProjectRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output[:len(output)-1]), nil
}
