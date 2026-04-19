// Package client is an HTTP client for the ai-critic server, suitable for
// use by CLIs and other Go programs. It mirrors the behaviour of the web
// frontend (chunked uploads, bearer-token auth, etc.).
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to an ai-critic server over HTTP.
type Client struct {
	// Server is the base URL, e.g. "https://host.example.com". Trailing
	// slashes are stripped automatically.
	Server string
	// Token is the bearer token used for auth. May be empty.
	Token string
	// HTTPClient is the underlying *http.Client. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// New constructs a Client, trimming trailing slashes from server.
func New(server string, token string) *Client {
	return &Client{
		Server: strings.TrimRight(server, "/"),
		Token:  token,
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// NewRequest builds an http.Request against the configured server with the
// Authorization header set when a token is configured.
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.Server+path, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return req, nil
}

// Do executes the request with the configured HTTP client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient().Do(req)
}

// CheckAuth verifies the server + token are valid by calling /api/auth/check.
func (c *Client) CheckAuth() error {
	req, err := c.NewRequest(http.MethodGet, "/api/auth/check", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

// HomeInfo is the response of /api/files/home.
type HomeInfo struct {
	Home string `json:"home"`
	Cwd  string `json:"cwd"`
}

// GetHome fetches the server's home directory and working directory.
func (c *Client) GetHome() (*HomeInfo, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/files/home", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	var out HomeInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode home response: %w", err)
	}
	if out.Home == "" {
		return nil, fmt.Errorf("server returned empty home dir")
	}
	return &out, nil
}

// readAPIError extracts a readable error from a non-2xx response.
func readAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("%s: %s", resp.Status, errResp.Error)
	}
	snippet := strings.TrimSpace(string(data))
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	if snippet == "" {
		return fmt.Errorf("%s", resp.Status)
	}
	return fmt.Errorf("%s: %s", resp.Status, snippet)
}
