package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	envpkg "github.com/xhd2015/lifelog-private/ai-critic/server/env"
)

const help = `Usage: go run ./script/debug-server-and-frontend [options] [script]

Starts a quick-test server with the latest code and opens a browser debugger.

This script runs quick-test (which manages vite and server) and opens a browser debugger for JS code evaluation.

Options:
  -h, --help        Show this help message
  --port PORT       Port for quick-test server (default: 3580)
  --headless        Run browser in headless mode
  --no-vite         Pass to quick-test: don't auto-start vite (use built frontend)
  --restart-exec    Use exec restart when port is in use (preserves PID, faster but riskier)

If script is omitted, a default script is used to open the root page and print the title.
`

const defaultDebugScript = "await navigate('/'); console.log('Page title:', await page.title());"

func main() {
	fmt.Println("DEBUG: Starting main.go")
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Println("DEBUG: main.go completed successfully")
}

func run(args []string) error {
	fmt.Println("DEBUG: run() called with args:", args)
	var opts lib.QuickTestOptions

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %v", err)
	}
	if err := envpkg.Load(); err != nil {
		return fmt.Errorf("failed to load env: %v", err)
	}
	opts.Local = os.Getenv(lib.EnvQuickTestDefaultConfig) == lib.QuickTestDefaultConfigLocal
	defaultHeadless := envBool("BROWSER_DEBUG_DEFAULT_HEADLESS")
	headless := defaultHeadless

	args, err = flags.
		Int("--port", &opts.Port).
		Bool("--headless", &headless).
		Bool("--no-vite", &opts.NoVite).
		Bool("--restart-exec", &opts.RestartExec).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	fmt.Println("DEBUG: args after parsing:", args)
	fmt.Println("DEBUG: opts.Port:", opts.Port, "headless:", headless, "restartExec:", opts.RestartExec)

	if len(args) > 1 {
		return fmt.Errorf("at most one script argument is supported")
	}
	scriptArg := defaultDebugScript
	if len(args) == 1 {
		scriptArg = args[0]
	}

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
		if result != nil && result.ServerCmd != nil && result.ServerCmd.Process != nil {
			result.ServerCmd.Process.Kill()
		}
		if result != nil && result.ViteCmd != nil && result.ViteCmd.Process != nil {
			result.ViteCmd.Process.Kill()
		}
		return fmt.Errorf("server failed to start: %v", err)
	}
	fmt.Println("Server is ready!")

	fmt.Println("Starting browser debugger...")
	debugCmd := exec.CommandContext(ctx, "go", "run", "./script/debug-port", fmt.Sprintf("--port=%d", opts.GetPort()))
	debugCmd.Args = append(debugCmd.Args, fmt.Sprintf("--headless=%v", headless))
	debugCmd.Args = append(debugCmd.Args, scriptArg)
	debugCmd.Dir = projectRoot
	debugCmd.Stdin = os.Stdin
	debugCmd.Stdout = os.Stdout
	debugCmd.Stderr = os.Stderr

	debugErr := debugCmd.Run()

	if result != nil && result.ServerCmd != nil && result.ServerCmd.Process != nil {
		fmt.Println("Stopping quick-test server...")
		result.ServerCmd.Process.Signal(syscall.SIGTERM)
		result.ServerCmd.Wait()
	}

	if result != nil && result.ViteCmd != nil && result.ViteCmd.Process != nil {
		fmt.Println("Stopping Vite dev server...")
		result.ViteCmd.Process.Signal(syscall.SIGTERM)
		result.ViteCmd.Process.Wait()
	}

	if debugErr != nil {
		if !headless {
			return fmt.Errorf(
				"failed to launch browser in non-headless mode: %v\n\nTry one of:\n  1) set BROWSER_DEBUG_DEFAULT_HEADLESS=true in .env/.env.local\n  2) pass --headless=true\n  3) pass --headless=false to force visible mode when your machine supports it",
				debugErr,
			)
		}
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

func envBool(key string) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
