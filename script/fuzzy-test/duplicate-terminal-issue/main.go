package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xhd2015/agent-pro/pkgs/containers/podman"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/less-gen/flags"
)

const help = `Usage: go run ./script/fuzzy-test [options]

Runs a fuzzy test that simulates random latency on the terminal sessions API
to detect duplicate terminal session creation bugs.

The test auto-starts the quick-test server, opens a headless browser, and
repeatedly navigates to /terminal and refreshes while injecting random delays
on API responses. It checks GET /api/terminal/sessions after each action;
if > 1 session exists, a duplicate bug is flagged.

Runs forever until Ctrl-C (or --max-iterations).

Options:
  -h, --help            Show this help message
  --port PORT           Server port (default: 3580)
  --headless            Run browser headless (default: true)
  --headless=false      Run browser visibly
  --max-iterations N    Max iterations, 0 = run forever (default: 0)
  --log-file PATH       Log output file (default: script/fuzzy-test/fuzzy-test-output.log)
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
	headless := true
	maxIterations := 0
	logFilePath := ""

	args, err := flags.
		Int("--port", &opts.Port).
		Bool("--headless", &headless).
		Int("--max-iterations", &maxIterations).
		String("--log-file", &logFilePath).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if opts.Port == 0 {
		opts.Port = lib.QuickTestPort
	}
	if logFilePath == "" {
		logFilePath = filepath.Join("script", "fuzzy-test", "fuzzy-test-output.log")
	}

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %v", err)
	}

	// Create log file (truncate on each run)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file %s: %v", logFilePath, err)
	}
	defer logFile.Close()

	// Tee output to both stdout and log file
	writer := io.MultiWriter(os.Stdout, logFile)

	opts.Stdout = writer
	opts.Stderr = writer
	opts.ProjectDir = projectRoot

	// ---- Start server ----
	fmt.Fprintf(writer, "[fuzzy] Building and starting server...\n")
	if err := lib.QuickTestPrepare(&opts); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := lib.QuickTestStart(ctx, &opts)
	if err != nil {
		return err
	}

	defer func() {
		if result.ServerCmd != nil && result.ServerCmd.Process != nil {
			fmt.Fprintf(writer, "[fuzzy] Stopping server...\n")
			result.ServerCmd.Process.Signal(syscall.SIGTERM)
			result.ServerCmd.Wait()
		}
		if result.ViteCmd != nil && result.ViteCmd.Process != nil {
			fmt.Fprintf(writer, "[fuzzy] Stopping Vite...\n")
			result.ViteCmd.Process.Signal(syscall.SIGTERM)
			result.ViteCmd.Wait()
		}
	}()

	// Wait for server ready
	port := opts.GetPort()
	fmt.Fprintf(writer, "[fuzzy] Waiting for server on port %d...\n", port)
	if err := waitForPort(ctx, port, 60*time.Second); err != nil {
		return fmt.Errorf("server failed to start: %v", err)
	}
	fmt.Fprintf(writer, "[fuzzy] Server is ready\n")

	// ---- Run Playwright fuzzy test ----
	fuzzyScriptPath := filepath.Join(projectRoot, "script", "fuzzy-test", "fuzzy.js")
	if _, err := os.Stat(fuzzyScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("fuzzy.js not found at %s", fuzzyScriptPath)
	}

	// Run node with NODE_PATH pointing to debug-port/node_modules where playwright lives
	debugPortDir := filepath.Join(projectRoot, "script", "debug-port")
	cmd := exec.CommandContext(ctx, "node", fuzzyScriptPath)
	cmd.Dir = debugPortDir
	cmd.Stdout = writer
	cmd.Stderr = writer

	baseURL := lib.QuickTestBaseURL(port)
	nodeModulesPath := filepath.Join(debugPortDir, "node_modules")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NODE_PATH=%s", nodeModulesPath),
		fmt.Sprintf("BASE_URL=%s", baseURL),
		fmt.Sprintf("HEADLESS=%v", headless),
		fmt.Sprintf("MAX_ITERATIONS=%d", maxIterations),
	)

	fmt.Fprintf(writer, "[fuzzy] Starting fuzzy test against %s\n", baseURL)
	fmt.Fprintf(writer, "[fuzzy] Headless: %v, Max iterations: %d (0=forever)\n", headless, maxIterations)
	fmt.Fprintf(writer, "[fuzzy] Press Ctrl-C to stop\n\n")

	// Signal handling: forward Ctrl-C to node process gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Run()
	}()

	select {
	case err := <-doneCh:
		if err != nil {
			fmt.Fprintf(writer, "[fuzzy] Test exited with error: %v\n", err)
		}
	case sig := <-sigCh:
		fmt.Fprintf(writer, "\n[fuzzy] Received signal %v, stopping...\n", sig)
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-doneCh:
			case <-time.After(10 * time.Second):
				cmd.Process.Kill()
			}
		}
	}

	fmt.Fprintf(writer, "\n[fuzzy] Log saved to %s\n", logFilePath)
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
		if podman.CheckPort(port) {
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
