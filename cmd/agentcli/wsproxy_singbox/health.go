package wsproxy_singbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

const (
	upstreamHealthInterval = time.Second
	upstreamDialTimeout    = 100 * time.Millisecond
)

// upstreamReachable is overridden in tests.
var upstreamReachable = defaultUpstreamReachable

func defaultUpstreamReachable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), upstreamDialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ProbeUpstreamProxy reports whether the local xray SOCKS port accepts TCP.
func ProbeUpstreamProxy(port int) bool {
	return upstreamReachable(port)
}

// switchWebOutbound is overridden in tests.
var switchWebOutbound = defaultSwitchWebOutbound

func defaultSwitchWebOutbound(outbound string) error {
	body, err := json.Marshal(map[string]string{"name": outbound})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("http://%s/proxies/%s", clashAPIListen, webSelectorTag), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("clash api PUT /proxies/%s: HTTP %d", webSelectorTag, resp.StatusCode)
	}
	return nil
}

// StartWebOutboundHealthMonitor toggles the web selector between proxy and direct.
func StartWebOutboundHealthMonitor(socksPort int) func() {
	ctx, cancel := context.WithCancel(context.Background())
	var usingProxy atomic.Bool
	usingProxy.Store(ProbeUpstreamProxy(socksPort))
	initial := directOutboundTag
	if usingProxy.Load() {
		initial = proxyOutboundTag
	}
	_ = switchWebOutbound(initial)

	go func() {
		ticker := time.NewTicker(upstreamHealthInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reachable := ProbeUpstreamProxy(socksPort)
				prev := usingProxy.Load()
				if reachable == prev {
					continue
				}
				usingProxy.Store(reachable)
				next := directOutboundTag
				if reachable {
					next = proxyOutboundTag
				}
				if err := switchWebOutbound(next); err != nil {
					fmt.Fprintf(os.Stderr, "warning: switch web outbound to %s: %v\n", next, err)
				}
			}
		}
	}()
	return cancel
}