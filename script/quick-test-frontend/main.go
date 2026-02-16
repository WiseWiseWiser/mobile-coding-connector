package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const (
	vitePort       = 5173
	defaultTimeout = 10 * time.Minute
)

const help = `
Usage: go run ./script/quick-test-frontend [options]

Starts a quick-test frontend environment for debugging:
- Vite dev server on port 57384
- Quick-test backend on port 37651
- Proxies frontend requests to backend
- Auto-shuts down after 10 minutes of inactivity

Options:
  --timeout DURATION   Auto-shutdown timeout (default: 10m)
  -h, --help          Show this help message
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

	timeout := defaultTimeout

	// Parse flags
	remainingArgs := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--timeout" && i+1 < len(args) {
			dur, err := time.ParseDuration(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid timeout: %v", err)
			}
			timeout = dur
			i++
			continue
		}
		remainingArgs = append(remainingArgs, arg)
	}

	if len(remainingArgs) > 0 {
		return fmt.Errorf("unrecognized args: %s", strings.Join(remainingArgs, " "))
	}

	// Check for existing processes on ports
	fmt.Printf("Checking for existing processes on ports %d and %d...\n", vitePort, lib.QuickTestPort)

	// Kill any existing process on quick-test port
	if pid, err := getPidOnPort(lib.QuickTestPort); err == nil && pid != 0 {
		fmt.Printf("Killing existing server on port %d (PID: %d)...\n", lib.QuickTestPort, pid)
		syscall.Kill(pid, syscall.SIGKILL)
		time.Sleep(500 * time.Millisecond)
	}

	// Build the backend server
	fmt.Println("Building Go server...")
	if err := runCmd("go", "build", "-o", "/tmp/ai-critic-quick", "./"); err != nil {
		return fmt.Errorf("failed to build: %v", err)
	}

	// Start vite dev server
	fmt.Printf("Starting Vite dev server on port %d...\n", vitePort)
	viteCmd := exec.Command("bun", "run", "dev")
	viteCmd.Dir = "ai-critic-react"
	viteCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", vitePort))
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr
	if err := viteCmd.Start(); err != nil {
		return fmt.Errorf("failed to start vite: %v", err)
	}

	// Wait for vite to be ready
	fmt.Print("Waiting for Vite server...")
	viteReady := false
	for i := 0; i < 60; i++ {
		if checkPort(vitePort) {
			viteReady = true
			break
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()
	if !viteReady {
		return fmt.Errorf("Vite server failed to start")
	}
	fmt.Printf("Vite server ready on port %d\n", vitePort)

	// Start the quick-test backend
	fmt.Printf("Starting quick-test backend on port %d...\n", lib.QuickTestPort)
	backendCmd := exec.Command("/tmp/ai-critic-quick", "--quick-test")
	backendCmd.Stdout = os.Stdout
	backendCmd.Stderr = os.Stderr
	if err := backendCmd.Start(); err != nil {
		return fmt.Errorf("failed to start backend: %v", err)
	}
	fmt.Printf("Backend started with PID: %d\n", backendCmd.Process.Pid)

	// Wait for backend to be ready
	fmt.Print("Waiting for backend...")
	backendReady := false
	for i := 0; i < 30; i++ {
		if checkPort(lib.QuickTestPort) {
			backendReady = true
			break
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()
	if !backendReady {
		return fmt.Errorf("Backend failed to start")
	}
	fmt.Printf("Backend ready on port %d\n", lib.QuickTestPort)

	// Start the proxy server that routes /api/* to backend
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Proxy to vite for non-API requests
		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = fmt.Sprintf("localhost:%d", vitePort)
			},
		}
		proxy.ServeHTTP(w, r)
	})

	// Create API proxy handler
	apiProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("localhost:%d", lib.QuickTestPort)
		},
	}

	// Custom handler that routes /api/* to backend
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiProxy.ServeHTTP(w, r)
		} else {
			// Serve from vite
			proxy := &httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = fmt.Sprintf("localhost:%d", vitePort)
				},
			}
			proxy.ServeHTTP(w, r)
		}
	})

	proxyServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", lib.DefaultServerPort),
		Handler: proxyHandler,
	}

	// Start proxy server
	go func() {
		fmt.Printf("Starting proxy server on port %d...\n", lib.DefaultServerPort)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Proxy server error: %v\n", err)
		}
	}()

	// Wait a bit for proxy to start
	time.Sleep(1 * time.Second)

	fmt.Println()
	fmt.Println("============================================")
	fmt.Println("Quick-test frontend environment is ready!")
	fmt.Println()
	fmt.Printf("  Frontend:  http://localhost:%d\n", lib.DefaultServerPort)
	fmt.Printf("  Vite:      http://localhost:%d\n", vitePort)
	fmt.Printf("  Backend:   http://localhost:%d\n", lib.QuickTestPort)
	fmt.Println()
	fmt.Printf("Auto-shutdown in %s\n", timeout)
	fmt.Println("Press Ctrl+C to stop manually")
	fmt.Println("============================================")
	fmt.Println()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Auto-shutdown timer
	timer := time.AfterFunc(timeout, func() {
		fmt.Printf("\n[auto] Timeout reached (%s), shutting down...\n", timeout)
		sigChan <- syscall.SIGTERM
	})

	// Reset timer on activity (simple approach: just let it run)
	// For more sophisticated activity detection, we'd need to track requests

	// Wait for signal
	<-sigChan
	timer.Stop()

	fmt.Println("\nShutting down...")
	proxyServer.Shutdown(context.Background())

	if viteCmd.Process != nil {
		viteCmd.Process.Kill()
	}
	if backendCmd.Process != nil {
		backendCmd.Process.Kill()
	}

	fmt.Println("Done!")
	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getPidOnPort(port int) (int, error) {
	cmd := exec.Command("lsof", "-t", "-i", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil {
		return 0, nil
	}
	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, nil
	}
	return strconv.Atoi(pidStr)
}
