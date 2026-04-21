package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// GitCloneRequest is the body of POST /api/git/clone. Kept in sync with
// the server's server/git.CloneRequest.
type GitCloneRequest struct {
	// Repo is the repository URL to clone. Required.
	Repo string `json:"repo"`
	// Dir is the target directory on the server. If empty, the server
	// clones into ~/<repo_base_name>.
	Dir string `json:"dir"`
	// PrivateKey is the raw contents of an SSH private key. If non-empty,
	// the server materializes it, points GIT_SSH_COMMAND at it for the
	// clone, and removes the file when the clone finishes.
	PrivateKey string `json:"private_key"`
	// HTTPSProxy is the value to export as https_proxy / HTTPS_PROXY for
	// the remote git process.
	HTTPSProxy string `json:"https_proxy"`
}

// GitRepoOpRequest is the body of POST /api/git/fetch and POST /api/git/pull.
// Dir is required and must be an existing git repository on the server.
type GitRepoOpRequest struct {
	Dir        string `json:"dir"`
	PrivateKey string `json:"private_key"`
	HTTPSProxy string `json:"https_proxy"`
}

// GitClone invokes 'git clone' on the server and streams stdout/stderr
// back via handler. See streamGit for the return-value contract.
func (c *Client) GitClone(req GitCloneRequest, handler ExecHandler) (int, error) {
	if req.Repo == "" {
		return 0, fmt.Errorf("git clone: repo must be set")
	}
	return c.streamGit("/api/git/clone", req, handler)
}

// GitCloneWithKeyFile is a convenience wrapper around GitClone that reads
// a local private-key file and passes its contents in the request. When
// keyFile is empty, GitClone is called with PrivateKey left blank.
func (c *Client) GitCloneWithKeyFile(req GitCloneRequest, keyFile string, handler ExecHandler) (int, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return 0, fmt.Errorf("read private key file: %w", err)
		}
		req.PrivateKey = string(data)
	}
	return c.GitClone(req, handler)
}

// GitFetch invokes 'git fetch' inside req.Dir on the server.
func (c *Client) GitFetch(req GitRepoOpRequest, handler ExecHandler) (int, error) {
	if req.Dir == "" {
		return 0, fmt.Errorf("git fetch: dir must be set")
	}
	return c.streamGit("/api/git/fetch", req, handler)
}

// GitFetchWithKeyFile reads a local private-key file and forwards its
// contents to the server before calling GitFetch.
func (c *Client) GitFetchWithKeyFile(req GitRepoOpRequest, keyFile string, handler ExecHandler) (int, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return 0, fmt.Errorf("read private key file: %w", err)
		}
		req.PrivateKey = string(data)
	}
	return c.GitFetch(req, handler)
}

// GitPull invokes 'git pull --ff-only' inside req.Dir on the server.
func (c *Client) GitPull(req GitRepoOpRequest, handler ExecHandler) (int, error) {
	if req.Dir == "" {
		return 0, fmt.Errorf("git pull: dir must be set")
	}
	return c.streamGit("/api/git/pull", req, handler)
}

// GitPullWithKeyFile reads a local private-key file and forwards its
// contents to the server before calling GitPull.
func (c *Client) GitPullWithKeyFile(req GitRepoOpRequest, keyFile string, handler ExecHandler) (int, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return 0, fmt.Errorf("read private key file: %w", err)
		}
		req.PrivateKey = string(data)
	}
	return c.GitPull(req, handler)
}

// streamGit POSTs body as JSON to path and consumes the NDJSON event
// stream defined in server/git. The returned exit code mirrors the
// remote git process's exit code on success. A non-zero remote exit
// code is NOT returned as a Go error; inspect the returned code. An
// error is returned only when the HTTP call itself failed, the server
// refused the request, the stream was truncated, or the server emitted
// an "error" event.
func (c *Client) streamGit(path string, body any, handler ExecHandler) (int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("git: marshal request: %w", err)
	}

	httpReq, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, readAPIError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var (
		exitCode  int
		sawExit   bool
		streamErr error
	)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev ExecEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return 0, fmt.Errorf("git: decode event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		switch ev.Type {
		case "exit":
			exitCode = ev.Code
			sawExit = true
		case "error":
			streamErr = fmt.Errorf("server error: %s", ev.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("git: read stream: %w", err)
	}
	if streamErr != nil {
		return 0, streamErr
	}
	if !sawExit {
		return 0, fmt.Errorf("git: stream ended without exit event")
	}
	return exitCode, nil
}
