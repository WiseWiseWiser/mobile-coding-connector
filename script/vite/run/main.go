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
)

var help = fmt.Sprintf(`
Usage: go run ./script/vite/run [options]

Starts the Vite frontend dev server on port %d.

Options:
  --host HOST   Bind to a specific host (default: localhost)
  --port PORT   Use a specific port (default: %d)
  -h, --help    Show this help message
`, lib.ViteDevPort, lib.ViteDevPort)

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var host string
	var port string
	_, err := flags.
		String("--host", &host).
		String("--port", &port).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	// Create context for managing the subprocess
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down Vite dev server...")
		cancel()
	}()

	// Build vite dev args
	viteArgs := []string{"run", "dev"}
	if host != "" || port != "" {
		viteArgs = append(viteArgs, "--")
		if host != "" {
			viteArgs = append(viteArgs, "--host", host)
		}
		if port != "" {
			viteArgs = append(viteArgs, "--port", port)
		}
	}

	fmt.Printf("Starting Vite dev server in ai-critic-react/...\n")
	if host != "" {
		fmt.Printf("Host: %s\n", host)
	}
	if port != "" {
		fmt.Printf("Port: %s\n", port)
	} else {
		fmt.Printf("Port: %d (default)\n", lib.ViteDevPort)
	}

	cmd := exec.CommandContext(ctx, "npm", viteArgs...)
	cmd.Dir = "ai-critic-react"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Vite dev server: %v", err)
	}

	// Wait for process to exit or context to be cancelled
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	case err := <-done:
		if err != nil {
			return fmt.Errorf("vite dev server exited with error: %v", err)
		}
	}

	return nil
}
