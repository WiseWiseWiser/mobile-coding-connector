package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
)

const help = `
Usage: go run ./script/run [options]

Options:
  --dir DIR     Set the initial directory for code review (defaults to current working directory)
  -h, --help    Show this help message
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var dirFlag string
	args, err := flags.
		String("--dir", &dirFlag).
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

	// Check if bun installed
	if _, err := exec.LookPath("bun"); err != nil {
		return fmt.Errorf("bun is not installed, install it from https://bun.sh/docs/installation")
	}

	// Check if ai-critic-react/node_modules exists
	if _, err := os.Stat("ai-critic-react/node_modules"); err != nil {
		fmt.Println("Installing frontend dependencies...")
		err := cmd.Debug().Dir("ai-critic-react").Run("bun", "install")
		if err != nil {
			return err
		}
	}

	// Build the Go server
	fmt.Println("Building Go server...")
	err = cmd.Debug().Run("go", "build", "-o", "/tmp/ai-critic", "./")
	if err != nil {
		return fmt.Errorf("failed to build Go server: %v", err)
	}

	// Start vite dev server in background
	fmt.Println("Starting Vite dev server...")
	viteCmd := exec.CommandContext(ctx, "bun", "run", "dev")
	viteCmd.Dir = "ai-critic-react"
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr
	if err := viteCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Vite dev server: %v", err)
	}

	// Wait for Vite to be ready
	fmt.Print("Waiting for Vite server to be ready")
	viteReady := false
	for i := 0; i < 30; i++ {
		if checkPort(5173) {
			viteReady = true
			break
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()

	if !viteReady {
		return fmt.Errorf("Vite server failed to start within timeout")
	}
	fmt.Println("Vite server is ready!")

	// Use --dir flag if provided, otherwise use current working directory
	targetDir := dirFlag
	if targetDir == "" {
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
	}

	// Build command args
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

	// Start the Go server in dev mode
	fmt.Println("Starting Go server in dev mode...")
	fmt.Printf("Initial directory: %s\n", targetDir)
	goServerCmd := exec.CommandContext(ctx, "/tmp/ai-critic", serverArgs...)
	goServerCmd.Stdout = os.Stdout
	goServerCmd.Stderr = os.Stderr
	goServerCmd.Stdin = os.Stdin
	if err := goServerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Go server: %v", err)
	}

	// Wait for either process to exit or context to be cancelled
	done := make(chan error, 2)
	go func() {
		done <- viteCmd.Wait()
	}()
	go func() {
		done <- goServerCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, kill processes
		if viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		if goServerCmd.Process != nil {
			goServerCmd.Process.Kill()
		}
	case err := <-done:
		// One process exited, cancel context to kill the other
		cancel()
		if err != nil {
			return fmt.Errorf("process exited with error: %v", err)
		}
	}

	return nil
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
