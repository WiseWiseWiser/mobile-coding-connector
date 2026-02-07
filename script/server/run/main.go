package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/xgo/support/cmd"
)

var help = fmt.Sprintf(`
Usage: go run ./script/server/run [options]

Builds and runs the Go server only, proxying frontend requests to http://localhost:%d/

Options:
  --dir DIR     Set the initial directory for code review (defaults to current working directory)
  -h, --help    Show this help message

Note: Make sure to start the frontend dev server separately:
  cd ai-critic-react && npm run dev
`, lib.ViteDevPort)

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var dirFlag string
	var debugFlag bool
	args, err := flags.
		String("--dir", &dirFlag).
		Bool("--debug", &debugFlag).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	// Create context for managing subprocesses
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals to gracefully shutdown subprocesses
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Build the Go server
	fmt.Println("Building Go server...")
	binary := "/tmp/ai-critic"	
	buildArgs := []string{ "build", "-o", binary}
	if debugFlag {
		buildArgs = append(buildArgs, "-gcflags=all=-N -l")
	}
	buildArgs = append(buildArgs, "./")
	err = cmd.Debug().Run("go", buildArgs...)
	if err != nil {
		return fmt.Errorf("failed to build Go server: %v", err)
	}

	// Use --dir flag if provided, otherwise use current working directory
	targetDir := dirFlag
	if targetDir == "" {
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
	}

	// Build command args - use --dev to enable proxy to frontend
	serverArgs := []string{"--dev", "--dir", targetDir}

	// Check for .config.local.json in the current directory (ai-critic)
	configFile := ".config.local.json"
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Found config file: %s\n", configFile)
		serverArgs = append(serverArgs, "--config-file", configFile)
	}

	// Check for rules directory
	rulesDir := "rules"
	if _, err := os.Stat(rulesDir); err == nil {
		fmt.Printf("Found rules directory: %s\n", rulesDir)
		serverArgs = append(serverArgs, "--rules-dir", rulesDir)
	}

	// Start the Go server in dev mode (proxies to localhost:ViteDevPort)
	fmt.Println("Starting Go server in dev mode...")
	fmt.Printf("Initial directory: %s\n", targetDir)
	fmt.Printf("Frontend requests will be proxied to http://localhost:%d/\n", lib.ViteDevPort)
	fmt.Println("Make sure the frontend dev server is running: cd ai-critic-react && npm run dev")


	runBianry := binary
	runArgs := serverArgs
	if debugFlag {
		runBianry = "kool"
		runArgs = []string{ "debug", binary}
		runArgs = append(runArgs, serverArgs...)
	}
	
	goServerCmd := exec.CommandContext(ctx, runBianry, runArgs...)
	goServerCmd.Stdout = os.Stdout
	goServerCmd.Stderr = os.Stderr
	goServerCmd.Stdin = os.Stdin
	if err := goServerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Go server: %v", err)
	}

	// Wait for process to exit or context to be cancelled
	done := make(chan error, 1)
	go func() {
		done <- goServerCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, kill process
		if goServerCmd.Process != nil {
			goServerCmd.Process.Kill()
		}
	case err := <-done:
		if err != nil {
			return fmt.Errorf("server exited with error: %v", err)
		}
	}

	return nil
}
