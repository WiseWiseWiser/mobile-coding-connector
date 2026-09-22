package cloudflareproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// APIClient talks to a running proxy's control plane and holds dial workers.
type APIClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func (c *APIClient) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *APIClient) doJSON(method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, rdr)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		var er struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &er)
		if er.Error != "" {
			return resp.StatusCode, fmt.Errorf("%s", er.Error)
		}
		return resp.StatusCode, fmt.Errorf("proxy %s", resp.Status)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// ListMappings returns mappings for this token.
func (c *APIClient) ListMappings() ([]MappingView, error) {
	var out struct {
		Mappings []MappingView `json:"mappings"`
	}
	_, err := c.doJSON(http.MethodGet, "/api/mappings", nil, &out)
	return out.Mappings, err
}

// AddMapping registers a hostname and returns dial_url.
func (c *APIClient) AddMapping(hostname string) (MappingView, error) {
	var out MappingView
	_, err := c.doJSON(http.MethodPost, "/api/mappings", map[string]string{"hostname": hostname}, &out)
	return out, err
}

// DeleteMapping removes by id and/or hostname.
func (c *APIClient) DeleteMapping(id, hostname string) (string, error) {
	q := url.Values{}
	if id != "" {
		q.Set("id", id)
	}
	if hostname != "" {
		q.Set("hostname", hostname)
	}
	var out struct {
		Deleted string `json:"deleted"`
	}
	_, err := c.doJSON(http.MethodDelete, "/api/mappings?"+q.Encode(), nil, &out)
	return out.Deleted, err
}

// PublishOpts configures a blocking add session.
type PublishOpts struct {
	Hostname string
	Echo     bool
	Forward  string
	Pool     int
	Stdout   io.Writer
	Stderr   io.Writer
	OnReady  func()
}

// Publish creates a mapping, dials a WS pool, and serves until ctx is done.
// On return it deletes the mapping.
func (c *APIClient) Publish(ctx context.Context, opts PublishOpts) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if !opts.Echo && strings.TrimSpace(opts.Forward) == "" {
		return fmt.Errorf("need --echo or --forward URL")
	}
	if opts.Echo && strings.TrimSpace(opts.Forward) != "" {
		return fmt.Errorf("--echo and --forward are mutually exclusive")
	}
	if opts.Echo {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		defer ln.Close()
		srv := &http.Server{Handler: http.HandlerFunc(echoHTTP)}
		go func() { _ = srv.Serve(ln) }()
		defer func() { _ = srv.Close() }()
		opts.Forward = "http://" + ln.Addr().String()
		opts.Echo = false
	}
	host := normalizeHostname(opts.Hostname)
	if host == "" {
		return fmt.Errorf("--hostname is required")
	}
	pool := opts.Pool
	if pool <= 0 {
		pool = DefaultPoolSize
	}

	view, err := c.AddMapping(host)
	if err != nil {
		return err
	}
	deleted := false
	defer func() {
		if deleted {
			return
		}
		if _, derr := c.DeleteMapping(view.ID, ""); derr == nil {
			fmt.Fprintf(opts.Stdout, "deleted %s\n", view.ID)
		}
	}()

	handler := "echo"
	if !opts.Echo {
		handler = "forward"
	}
	fmt.Fprintf(opts.Stdout, "hostname:   %s\n", view.Hostname)
	fmt.Fprintf(opts.Stdout, "public:     %s\n", view.URL)
	fmt.Fprintf(opts.Stdout, "handler:    %s\n", handler)

	var connected atomic.Int64
	ready := make(chan struct{})
	var readyOnce sync.Once
	errCh := make(chan error, 1)

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < pool; i++ {
		go func() {
			backoff := 200 * time.Millisecond
			for {
				if workerCtx.Err() != nil {
					return
				}
				err := c.serveDial(workerCtx, view.DialURL, opts, &connected, func() {
					readyOnce.Do(func() { close(ready) })
				})
				if workerCtx.Err() != nil {
					return
				}
				if err != nil && isMappingClosed(err) {
					select {
					case errCh <- fmt.Errorf("mapping closed"):
					default:
					}
					cancel()
					return
				}
				t := time.NewTimer(backoff)
				select {
				case <-workerCtx.Done():
					t.Stop()
					return
				case <-t.C:
				}
				if backoff < 5*time.Second {
					backoff *= 2
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-ready:
		fmt.Fprintf(opts.Stdout, "ws:         connected  pool=%d\n", connected.Load())
		if opts.OnReady != nil {
			opts.OnReady()
		}
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for websocket")
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func isMappingClosed(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unauthorized") || strings.Contains(s, "mapping not found") || strings.Contains(s, "bad handshake")
}

func (c *APIClient) serveDial(ctx context.Context, dial string, opts PublishOpts, connected *atomic.Int64, onReady func()) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, dial, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("unauthorized")
		}
		return err
	}
	defer conn.Close()
	connected.Add(1)
	onReady()
	defer connected.Add(-1)

	addr, err := forwardTCPAddr(opts.Forward)
	if err != nil {
		return err
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	tcp, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer tcp.Close()

	pipe := newWSNetConn(conn)
	errc := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(tcp, pipe)
		errc <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(pipe, tcp)
		errc <- copyErr
	}()
	select {
	case <-ctx.Done():
		_ = tcp.Close()
		_ = pipe.Close()
		return ctx.Err()
	case err := <-errc:
		_ = tcp.Close()
		_ = pipe.Close()
		return err
	}
}

func forwardTCPAddr(forward string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(forward))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("bad --forward URL")
	}
	host := u.Host
	if u.Port() == "" {
		if u.Scheme == "https" {
			host = net.JoinHostPort(u.Hostname(), "443")
		} else {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
	}
	return host, nil
}

func echoHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	host := r.Host
	path := r.URL.RequestURI()
	if path == "" {
		path = "/"
	}
	fmt.Fprintf(w, "cloudflare-proxy echo\nmethod: %s\npath:   %s\nhost:   %s\n", r.Method, path, host)
}
