package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
)

var (
	proxySessMu sync.Mutex
	proxySess   = map[string]*proxySession{}
)

func proxyModeEnabled(cfg *CloudflareConfig) bool {
	return cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.Mode), "proxy")
}

// proxySession is one domain's live publish. The pointer is the identity: the
// publisher clears its own entry only when it is still the current one, so a
// restart cannot delete its successor's registration.
type proxySession struct {
	cancel context.CancelFunc
	// lastErr records why the publish ended, for the caller that already
	// returned successfully and would otherwise never learn about it.
	lastErr error
}

func proxySessionActive(domain string) bool {
	proxySessMu.Lock()
	defer proxySessMu.Unlock()
	_, ok := proxySess[domain]
	return ok
}

func stopProxySession(domain string) bool {
	proxySessMu.Lock()
	sess, ok := proxySess[domain]
	if ok {
		delete(proxySess, domain)
	}
	proxySessMu.Unlock()
	if ok && sess != nil && sess.cancel != nil {
		sess.cancel()
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
	sess := &proxySession{cancel: cancel}
	cli := &cloudflareproxy.APIClient{BaseURL: strings.TrimSpace(cfg.ProxyURL), Token: strings.TrimSpace(cfg.Token)}
	logFn(fmt.Sprintf("proxy mode: %s -> %s origin :%d", domain, cfg.ProxyURL, port))
	go func() {
		// Register from inside the publisher, so registration and cleanup
		// happen in one goroutine and cannot interleave. Previously the caller
		// registered the session and nothing ever removed it: when a publish
		// died after becoming ready, the registry kept claiming a session that
		// no longer existed, so status lied and no start path would republish.
		err := cli.Publish(ctx, cloudflareproxy.PublishOpts{
			Hostname: domain,
			Forward:  fmt.Sprintf("http://127.0.0.1:%d", port),
			Pool:     cloudflareproxy.DefaultPoolSize,
			Stdout:   logWriter(logFn),
			Stderr:   logWriter(logFn),
			OnReady: func() {
				proxySessMu.Lock()
				proxySess[domain] = sess
				proxySessMu.Unlock()
				readyOnce.Do(func() { close(ready) })
			},
		})
		proxySessMu.Lock()
		sess.lastErr = err
		if proxySess[domain] == sess {
			delete(proxySess, domain)
		}
		proxySessMu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			logFn(fmt.Sprintf("proxy publish ended for %s: %v", domain, err))
		}
		select {
		case errCh <- err:
		default:
		}
	}()
	select {
	case <-ready:
		// The publisher registers the session as it becomes ready, so a publish
		// that died in the meantime has already removed it.
		proxySessMu.Lock()
		current := proxySess[domain]
		ended := sess.lastErr
		proxySessMu.Unlock()
		if current != sess {
			cancel()
			if ended == nil {
				ended = fmt.Errorf("proxy publish ended before it could serve")
			}
			return nil, ended
		}
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
