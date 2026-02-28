//go:build ignore

// Bug reproduction: HTTP server blocking when git fetch runs without SSH key
// This replicates the main server behavior using gitrunner, sse, and cloudflare tunnel packages.
package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	cloudflareSettings "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

const (
	serverPort   = 8080
	tunnelDomain = "test-bug-hang.xhd2015.xyz"
)

func main() {
	var requestCount int

	// Health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		fmt.Fprintf(w, "OK #%d - %s\n", requestCount, time.Now().Format(time.RFC3339))
	})

	// Git fetch endpoint with SSE streaming (like the real server)
	http.HandleFunc("/api/git/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		repo := r.URL.Query().Get("repo")
		if repo == "" {
			repo = "/root/lifelog-private"
		}

		// Set up SSE streaming (like server/github/gitops.go)
		sw := sse.NewWriter(w)
		if sw == nil {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		sw.SendLog(fmt.Sprintf("Starting git fetch in %s", repo))

		// Build git command using gitrunner (like the real server)
		cmd := gitrunner.Fetch().Dir(repo).Exec()

		sw.SendLog(fmt.Sprintf("$ git fetch %s", repo))

		// Stream command output
		cmdErr := sw.StreamCmd(cmd)
		if cmdErr != nil {
			sw.SendError(fmt.Sprintf("git fetch failed: %v", cmdErr))
			return
		}

		sw.SendDone(map[string]string{
			"message": "git fetch completed successfully",
		})
	})

	// Direct fetch without SSE (for comparison)
	http.HandleFunc("/fetch-direct", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get("repo")
		if repo == "" {
			repo = "/root/lifelog-private"
		}

		fmt.Fprintf(w, "Starting git fetch in %s...\n", repo)

		// Use gitrunner directly
		output, err := gitrunner.Fetch().Dir(repo).Run()

		if err != nil {
			fmt.Fprintf(w, "Error: %v\nOutput: %s\n", err, string(output))
		} else {
			fmt.Fprintf(w, "Success!\n")
		}
	})

	// Fetch without gitrunner (raw exec)
	http.HandleFunc("/fetch-raw", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get("repo")
		if repo == "" {
			repo = "/root/lifelog-private"
		}

		fmt.Fprintf(w, "Starting raw git fetch in %s...\n", repo)

		// Raw exec without gitrunner environment setup
		cmd := exec.Command("git", "-C", repo, "fetch", "origin")

		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Fprintf(w, "Error: %v\nOutput: %s\n", err, string(output))
		} else {
			fmt.Fprintf(w, "Success!\n")
		}
	})

	// Multiple concurrent fetches
	http.HandleFunc("/fetch-many", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get("repo")
		if repo == "" {
			repo = "/root/lifelog-private"
		}

		count := 5
		fmt.Fprintf(w, "Launching %d concurrent git fetches to %s...\n", count, repo)

		// Launch multiple fetches
		for i := 0; i < count; i++ {
			go func(id int) {
				cmd := gitrunner.Fetch().Dir(repo).Exec()
				_, _ = cmd.CombinedOutput()
			}(i)
		}

		fmt.Fprintf(w, "Started %d fetches in background\n", count)
	})

	// Start Cloudflare tunnel (like main server)
	fmt.Printf("[tunnel] Auto-starting Cloudflare tunnel for %s...\n", tunnelDomain)
	go func() {
		logFn := func(msg string) {
			fmt.Printf("[tunnel] %s\n", msg)
		}
		_, err := cloudflareSettings.StartDomainTunnel(tunnelDomain, serverPort, "", logFn)
		if err != nil {
			fmt.Printf("[tunnel] Failed to start tunnel: %v\n", err)
		} else {
			fmt.Printf("[tunnel] Tunnel started successfully for %s\n", tunnelDomain)
		}
	}()

	fmt.Printf("Server starting on :%d\n", serverPort)
	fmt.Printf("Tunnel domain: %s\n", tunnelDomain)
	fmt.Println("")
	fmt.Println("Test endpoints:")
	fmt.Println("  GET  /                    - Health check")
	fmt.Println("  POST /api/git/fetch       - Git fetch with SSE streaming (like real server)")
	fmt.Println("  GET  /fetch-direct        - Direct git fetch with gitrunner")
	fmt.Println("  GET  /fetch-raw           - Raw git fetch without gitrunner")
	fmt.Println("  GET  /fetch-many          - Multiple concurrent fetches")
	fmt.Println("")
	fmt.Println("Test commands:")
	fmt.Printf("  curl http://localhost:%d/\n", serverPort)
	fmt.Printf("  curl -X POST http://localhost:%d/api/git/fetch\n", serverPort)
	fmt.Printf("  curl http://localhost:%d/fetch-direct\n", serverPort)
	fmt.Printf("  curl http://localhost:%d/fetch-raw\n", serverPort)
	fmt.Println("")
	fmt.Println("If the bug exists, requests after git fetch will hang.")

	if err := http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
