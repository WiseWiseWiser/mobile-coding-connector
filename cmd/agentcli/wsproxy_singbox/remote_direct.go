package wsproxy_singbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

// pushRemoteDirect PUTs freedom-egress patterns to the server before TUN start.
// A 404 means the server build is too old to honor --remote-direct.
func pushRemoteDirect(getClient func() (*client.Client, error), patterns []AlsoProxyPattern) error {
	raws := make([]string, 0, len(patterns))
	for _, p := range patterns {
		raws = append(raws, p.Raw)
	}
	body, err := json.Marshal(map[string]any{"patterns": raws})
	if err != nil {
		return err
	}

	c, err := getClient()
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPut, "/api/ws-proxy/remote-direct", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("Pushing %d --remote-direct pattern(s) to server...\n", len(raws))
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("remote-direct API: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("server does not support --remote-direct (update remote-agent server); PUT /api/ws-proxy/remote-direct returned 404")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("remote-direct API failed: %s", msg)
	}

	var out struct {
		Patterns []string `json:"patterns"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("remote-direct API: invalid response: %w", err)
	}
	fmt.Printf("Server remote_direct set: %v\n", out.Patterns)
	return nil
}
