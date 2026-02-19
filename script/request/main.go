package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const cookieName = lib.CookieName

var help = fmt.Sprintf(`Usage: go run ./script/request <path> [body] [options]

Sends an HTTP request to the local server at http://localhost:%d.
Automatically includes auth cookie from %s.

Arguments:
  path    API path (e.g. /api/checkpoints?project=myproject)
  body    Optional JSON body; if provided, sends POST; otherwise sends GET

Options:
  --port PORT  Port to use (defaults to %d)
  -h, --help   Show this help message

Examples:
  go run ./script/request /api/checkpoints?project=lifelog-private
  go run ./script/request /api/checkpoints '{"project_dir":"/path","name":"test","file_paths":["a.txt"]}'
  go run ./script/request /api/auth/check
  go run ./script/request --port 37651 /api/server/status
`, lib.DefaultServerPort, lib.CredentialsFile, lib.DefaultServerPort)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var port int
	args, err := flags.
		Int("--port", &port).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}

	if port == 0 {
		port = lib.DefaultServerPort
	}

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
	token, err := lib.LoadFirstTokenFromHome()
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
