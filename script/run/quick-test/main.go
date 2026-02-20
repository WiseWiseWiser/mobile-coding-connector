package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/run quick-test [options]

Options:
  -h, --help               Show this help message
  --keep                   Keep server running indefinitely (disable auto-shutdown)
  --no-vite                Don't auto-start vite (serve static frontend instead)
  --frontend-port PORT     Proxy frontend to PORT (assumes vite/frontend started externally)
  --port PORT              Port to run on (default: 3580)
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var opts lib.QuickTestOptions

	args, err := flags.
		Bool("--keep", &opts.Keep).
		Bool("--no-vite", &opts.NoVite).
		Int("--frontend-port", &opts.FrontendPort).
		Int("--port", &opts.Port).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unknown args: %v", args)
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

	err = lib.QuickTestPrepare(&opts)
	if err != nil {
		return err
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr

	result, err := lib.QuickTestStart(ctx, &opts)
	if err != nil {
		return err
	}

	if result.Restarted {
		fmt.Println("Server restarted successfully (PID preserved).")
		fmt.Println("Press Ctrl+C to stop manually.")
		<-ctx.Done()
		return nil
	}

	fmt.Printf("Server started with PID: %d\n", result.ServerCmd.Process.Pid)
	if opts.Keep {
		fmt.Println("Server will keep running indefinitely (--keep enabled).")
	} else {
		fmt.Println("Server will exit after 10 minutes of inactivity.")
	}
	fmt.Println("Press Ctrl+C to stop manually.")

	err = result.ServerCmd.Wait()

	if result.ViteCmd != nil && result.ViteCmd.Process != nil {
		fmt.Println("Stopping Vite dev server...")
		result.ViteCmd.Process.Signal(syscall.SIGTERM)
		result.ViteCmd.Wait()
	}

	if err != nil {
		return fmt.Errorf("server exited with error: %v", err)
	}
	return nil
}
