package cloudflareproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	return c.doJSONContext(context.Background(), method, path, body, out)
}

// doJSONContext is doJSON with a caller-supplied deadline.
func (c *APIClient) doJSONContext(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, rdr)
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
	return c.ListMappingsContext(context.Background())
}

// ListMappingsContext is ListMappings with a caller-supplied deadline, so a
// status reader cannot block on an unreachable edge.
func (c *APIClient) ListMappingsContext(ctx context.Context) ([]MappingView, error) {
	var out struct {
		Mappings []MappingView `json:"mappings"`
	}
	_, err := c.doJSONContext(ctx, http.MethodGet, "/api/mappings", nil, &out)
	return out.Mappings, err
}

// AddMapping registers a hostname and returns dial_url.
//
// A previous process can leave its mapping behind: an exec-restart replaces the
// process image, so shutdown handlers never delete it. Registering the hostname
// again then fails with 409 "hostname already mapped", which used to abort
// proxy-mode auto-start and leave every domain at 503 until an operator deleted
// the mapping by hand.
//
// The hostname belongs to this origin, so an existing mapping with no live dial
// sockets is an orphan and is reclaimed. A mapping that still has sockets is
// owned by a running origin, and the conflict is reported instead.
func (c *APIClient) AddMapping(hostname string) (MappingView, error) {
	var out MappingView
	status, err := c.doJSON(http.MethodPost, "/api/mappings", map[string]string{"hostname": hostname}, &out)
	if err == nil {
		return out, nil
	}
	if status != http.StatusConflict {
		return out, err
	}

	reclaimed, reclaimErr := c.reclaimDeadMapping(hostname)
	if reclaimErr != nil {
		return out, fmt.Errorf("%w (could not reclaim the stale mapping: %v)", err, reclaimErr)
	}
	if !reclaimed {
		return out, fmt.Errorf("%w (a live origin is still publishing this hostname)", err)
	}

	var retry MappingView
	if _, err := c.doJSON(http.MethodPost, "/api/mappings", map[string]string{"hostname": hostname}, &retry); err != nil {
		return retry, err
	}
	return retry, nil
}

// reclaimDeadMapping deletes an orphaned mapping for hostname, reporting
// whether it reclaimed one. A mapping with live dial sockets is left alone.
func (c *APIClient) reclaimDeadMapping(hostname string) (bool, error) {
	mappings, err := c.ListMappings()
	if err != nil {
		return false, err
	}
	for _, mapping := range mappings {
		if mapping.Hostname != hostname {
			continue
		}
		if mapping.WS > 0 {
			return false, nil
		}
		if _, err := c.DeleteMapping(mapping.ID, hostname); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
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

// Dial pool refill timing. Each dial socket serves exactly one request before
// the origin closes it, so a worker that just served a request must be able to
// refill almost immediately; the growing backoff is only for repeated failures.
const (
	minDialBackoff = 200 * time.Millisecond
	maxDialBackoff = 5 * time.Second
)

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
			backoff := minDialBackoff
			for {
				if workerCtx.Err() != nil {
					return
				}
				served, err := c.serveDial(workerCtx, view.DialURL, opts, &connected, func() {
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
				// A dial that reached the origin is refilled without backoff. The
				// pool is a ready-queue of single-use sockets, so delay here is
				// what starves visitors into 503s.
				if served {
					backoff = minDialBackoff
				}
				t := time.NewTimer(backoff)
				select {
				case <-workerCtx.Done():
					t.Stop()
					return
				case <-t.C:
				}
				if backoff < maxDialBackoff {
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

// errMappingClosed marks a dial whose mapping can never serve again: the edge
// no longer knows the id, or rejects its dial token. Unlike a transport
// failure, retrying the same dial URL is pointless, so the session ends.
var errMappingClosed = errors.New("mapping closed")

// isMappingClosed reports whether the pool should stop rather than retry.
//
// The edge answers 401 on /dial both for an unknown mapping id and for a bad
// dial token, and that dial URL can never serve again — so 401 is terminal.
// Every other failure (a gateway 5xx, a timeout, a reset) is a transport blip
// on a still-valid URL and must be retried. Classifying these was previously
// done by matching error text, which treated the generic "websocket: bad
// handshake" that gorilla returns for *any* non-101 response — including a
// routine Cloudflare 502 — as a closed mapping. One such response cancelled the
// hostname's whole pool and silently killed its publish session, leaving the
// edge with no dial sockets ("no connected origin") until an operator
// republished by hand.
func isMappingClosed(err error) bool {
	return errors.Is(err, errMappingClosed)
}

// dialFailure converts a failed /dial handshake into the error the pool loop
// classifies. A 401 response is terminal; anything else stays retryable.
func dialFailure(resp *http.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		return errMappingClosed
	}
	return err
}

// serveDial connects one edge dial socket and pipes it to the origin.
//
// The origin connection is opened lazily, after the first request bytes arrive.
// An idle pooled dial must not hold an origin connection, because the origin
// HTTP server's ReadHeaderTimeout reaps connections that have not sent headers
// yet, which silently turns the edge's pooled socket into a corpse. The bool
// reports whether the pipe was established, so the caller can refill the pool
// without backoff once a socket has served its request.
func (c *APIClient) serveDial(ctx context.Context, dial string, opts PublishOpts, connected *atomic.Int64, onReady func()) (bool, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, dial, nil)
	if err != nil {
		return false, dialFailure(resp, err)
	}
	defer conn.Close()
	connected.Add(1)
	onReady()
	defer connected.Add(-1)

	addr, err := forwardTCPAddr(opts.Forward)
	if err != nil {
		return false, err
	}
	pipe := newWSNetConn(conn)

	// Wait for the edge's first request bytes; the socket reader answers the
	// edge's keepalive pings while this waits.
	head, err := readFirstChunk(ctx, pipe)
	if err != nil {
		return false, err
	}

	d := net.Dialer{Timeout: 10 * time.Second}
	tcp, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	defer tcp.Close()
	if _, err := tcp.Write(head); err != nil {
		return false, err
	}

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
		return true, ctx.Err()
	case err := <-errc:
		_ = tcp.Close()
		_ = pipe.Close()
		return true, err
	}
}

// readFirstChunk blocks until the edge sends the first request bytes, the dial
// dies, or ctx ends.
func readFirstChunk(ctx context.Context, pipe *wsNetConn) ([]byte, error) {
	type readResult struct {
		data []byte
		err  error
	}
	results := make(chan readResult, 1)
	go func() {
		head := make([]byte, 0, 4096)
		chunk := make([]byte, 32*1024)
		for {
			n, readErr := pipe.Read(chunk)
			if n > 0 {
				head = append(head, chunk[:n]...)
				results <- readResult{data: head}
				return
			}
			if readErr != nil {
				results <- readResult{err: readErr}
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		_ = pipe.Close()
		return nil, ctx.Err()
	case r := <-results:
		return r.data, r.err
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
