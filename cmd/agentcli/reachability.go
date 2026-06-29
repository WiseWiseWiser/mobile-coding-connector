package agentcli

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
)

func checkLocalServerReachable(server string) error {
	forced, up := testhooks.ReachabilityForced()
	if forced {
		if up {
			return nil
		}
		return notListeningError(server)
	}

	u, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("invalid server URL %q: %w", server, err)
	}
	host := u.Hostname()
	if host == "localhost" {
		host = "127.0.0.1"
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	addr := net.JoinHostPort(host, port)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return notListeningError(server)
	}
	_ = conn.Close()

	pingBase := strings.TrimRight(server, "/")
	if strings.Contains(pingBase, "://localhost") {
		pingBase = strings.Replace(pingBase, "://localhost", "://127.0.0.1", 1)
	}
	pingURL := pingBase + "/ping"
	client := &http.Client{Timeout: 3 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 15; attempt++ {
		resp, err := client.Get(pingURL)
		if err != nil {
			lastErr = err
			time.Sleep(200 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "pong" {
			return nil
		}
		lastErr = fmt.Errorf("unexpected ping response")
		time.Sleep(200 * time.Millisecond)
	}
	_ = lastErr
	return notListeningError(server)
}

func notListeningError(server string) error {
	return fmt.Errorf("server at %s is not listening\n\nStart the server with: ai-critic", server)
}