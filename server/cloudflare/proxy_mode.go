package cloudflare

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
)

var (
	proxySessMu sync.Mutex
	proxySess   = map[string]context.CancelFunc{}
)

func proxyModeEnabled(cfg *CloudflareConfig) bool {
	return cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.Mode), "proxy")
}

func proxySessionActive(domain string) bool {
	proxySessMu.Lock()
	defer proxySessMu.Unlock()
	_, ok := proxySess[domain]
	return ok
}

func stopProxySession(domain string) bool {
	proxySessMu.Lock()
	cancel, ok := proxySess[domain]
	if ok {
		delete(proxySess, domain)
	}
	proxySessMu.Unlock()
	if ok && cancel != nil {
		cancel()
		return true
	}
	return false
}

func startViaProxy(domain string, port int, cfg *CloudflareConfig, logFn LogFunc) (*DomainTunnelStatus, error) {
	if logFn == nil {
		logFn = func(string) {}
	}
	if strings.TrimSpace(cfg.ProxyURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("cloudflare mode=proxy requires proxy_url and token")
	}
	stopProxySession(domain)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	var readyOnce sync.Once
	errCh := make(chan error, 1)
	cli := &cloudflareproxy.APIClient{BaseURL: strings.TrimSpace(cfg.ProxyURL), Token: strings.TrimSpace(cfg.Token)}
	logFn(fmt.Sprintf("proxy mode: %s -> %s origin :%d", domain, cfg.ProxyURL, port))
	go func() {
		err := cli.Publish(ctx, cloudflareproxy.PublishOpts{
			Hostname: domain,
			Forward:  fmt.Sprintf("http://127.0.0.1:%d", port),
			Pool:     cloudflareproxy.DefaultPoolSize,
			Stdout:   logWriter(logFn),
			Stderr:   logWriter(logFn),
			OnReady: func() {
				readyOnce.Do(func() { close(ready) })
			},
		})
		select {
		case errCh <- err:
		default:
		}
	}()
	select {
	case <-ready:
		proxySessMu.Lock()
		proxySess[domain] = cancel
		proxySessMu.Unlock()
		logFn("proxy websocket connected")
		return &DomainTunnelStatus{Status: "active", TunnelURL: fmt.Sprintf("https://%s", domain)}, nil
	case err := <-errCh:
		cancel()
		if err == nil {
			err = fmt.Errorf("proxy publish ended before ready")
		}
		return nil, err
	case <-time.After(20 * time.Second):
		cancel()
		return nil, fmt.Errorf("timeout waiting for proxy websocket")
	}
}

type logWriterFunc func(string)

func logWriter(fn LogFunc) io.Writer {
	return logWriterFunc(fn)
}

func (f logWriterFunc) Write(p []byte) (int, error) {
	s := strings.TrimRight(string(p), "\n")
	if s != "" {
		f(s)
	}
	return len(p), nil
}
