package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

const cookieName = "ai-critic-token"

var credentialsFile = config.CredentialsFile

var help = fmt.Sprintf(`Usage: go run ./script/request <path> [body]

Sends an HTTP request to the local server at http://localhost:%d.
Automatically includes auth cookie from %s.

Arguments:
  path    API path (e.g. /api/checkpoints?project=myproject)
  body    Optional JSON body; if provided, sends POST; otherwise sends GET
  --port  Port to use (defaults to %d)

Examples:
  go run ./script/request /api/checkpoints?project=lifelog-private
  go run ./script/request /api/checkpoints '{"project_dir":"/path","name":"test","file_paths":["a.txt"]}'
  go run ./script/request /api/auth/check
  go run ./script/request --port 37651 /api/server/status
`, lib.DefaultServerPort, credentialsFile, lib.DefaultServerPort)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(help)
		return nil
	}

	// Parse --port flag
	port := lib.DefaultServerPort
	remainingArgs := []string{}
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--port" && i+1 < len(args) {
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid port: %s", args[i+1])
			}
			port = p
			i += 2
			continue
		}
		remainingArgs = append(remainingArgs, arg)
		i++
	}
	args = remainingArgs

	path := args[0]
	body := ""
	if len(args) > 1 {
		body = args[1]
	}

	// Build URL
	url := fmt.Sprintf("http://localhost:%d%s", port, path)

	// Determine HTTP method
	method := http.MethodGet
	var bodyReader io.Reader
	if body != "" {
		method = http.MethodPost
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Load auth token from credentials file
	token, err := loadFirstToken()
	if err == nil && token != "" {
		req.AddCookie(&http.Cookie{
			Name:  cookieName,
			Value: token,
		})
	}

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Print status
	fmt.Fprintf(os.Stderr, "%s %s → %s\n", method, path, resp.Status)

	// Print response body
	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	fmt.Println()

	return nil
}

// loadFirstToken reads the first non-empty line from the credentials file.
func loadFirstToken() (string, error) {
	f, err := os.Open(credentialsFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", scanner.Err()
}
