package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const requestHelp = `Usage: remote-agent request <api-path> [json-body]

Call an arbitrary API endpoint on the configured remote-agent server.
With no body, the request uses GET. With json-body, the request uses
POST and sends the body as application/json. If json-body is omitted
but stdin is piped, non-empty stdin is used as the JSON body.

Arguments:
  api-path      API path on the remote server, e.g. /api/services.
  json-body     Optional JSON request body. Use '-' to force reading
                from stdin.

Examples:
  remote-agent request /api/services
  remote-agent request /api/services/start?id=svc-123 '{}'
  echo '{"name":"demo"}' | remote-agent request /api/some
`

func runRequest(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.
		Help("-h,--help", requestHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("request requires <api-path> [json-body]; see 'remote-agent request --help'")
	}
	if len(args) > 2 {
		return fmt.Errorf("request takes at most 2 positional arguments, got %d", len(args))
	}

	path := strings.TrimSpace(args[0])
	if path == "" {
		return fmt.Errorf("api-path cannot be empty")
	}
	if strings.Contains(path, "://") {
		return fmt.Errorf("api-path must be a path on the configured server, not a full URL")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var body []byte
	method := http.MethodGet
	if len(args) == 2 {
		method = http.MethodPost
		if args[1] == "-" {
			body, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read request body from stdin: %w", err)
			}
		} else {
			body = []byte(args[1])
		}
		if err := validateJSONBody(body); err != nil {
			return err
		}
	} else {
		body, err = readPipedRequestBody()
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(body)) > 0 {
			method = http.MethodPost
			if err := validateJSONBody(body); err != nil {
				return err
			}
		} else {
			body = nil
		}
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := cli.NewRequest(method, path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("read response body: %w", readErr)
	}
	if len(data) > 0 {
		if _, err := os.Stdout.Write(data); err != nil {
			return fmt.Errorf("write response body: %w", err)
		}
		if data[len(data)-1] != '\n' {
			fmt.Println()
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

func validateJSONBody(body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return fmt.Errorf("json-body cannot be empty")
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("json-body must be valid JSON: %w", err)
	}
	return nil
}

func readPipedRequestBody() ([]byte, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}
	mode := info.Mode()
	if mode&os.ModeCharDevice != 0 {
		return nil, nil
	}
	if mode.IsRegular() && info.Size() == 0 {
		return nil, nil
	}
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("read request body from stdin: %w", err)
	}
	return body, nil
}
