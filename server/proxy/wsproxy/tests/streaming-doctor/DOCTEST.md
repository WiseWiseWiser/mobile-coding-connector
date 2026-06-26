# Streaming Doctor Doctests

Server-side unit tests for the ws-proxy doctor SSE stream and the shared
`server/streaming/progress` writer.

# DSN (Domain Specific Notion)

The streaming doctor harness models incremental health-check reporting from the
ai-critic server to any SSE consumer.

**Participants**

- **DoctorStream handler** — `GET /api/ws-proxy/doctor/stream`; runs
  `serverDoctorChecks` and emits each `DoctorCheck` as it completes.
- **progress.Writer** — thin wrapper over `sse.Writer`; emits typed JSON frames
  (`progress`, `section`, `meta`, `done`, `error`).
- **Manager** — loads ws-proxy config, probes local xray, evaluates tunnel
  ingress, optionally runs upstream fetch (stubbed in tests).
- **httptest.ResponseRecorder** — captures the SSE body for parsing assertions.

**Behaviors**

- Each `add()` during `serverDoctorChecks` becomes an immediate `progress` event;
  checks are not batched into the terminal `done` frame.
- `done` carries aggregate health (`healthy`, `try_url`, `checks_total`,
  `checks_failed`).
- Slow checks (e.g. `upstream_fetch`) interleave with fast checks when a test
  hook injects delay — proving incremental streaming.
- `progress.Writer` sets `Content-Type: text/event-stream` and flushes after
  every frame.

## Version

0.0.2

## Decision Tree

```
[streaming doctor]
 |
 +-- progress-writer/                         (grouping — framework only)
 |    |
 |    +-- emits-ordered-sse-frames/           (LEAF)  2 progress + section + done
 |    +-- content-type-event-stream/           (LEAF)  SSE headers on first byte
 |
 +-- doctor-stream/                           (grouping — Manager.DoctorStream)
      |
      +-- server-checks-emit-in-order/        (LEAF)  progress ids match legacy order
      +-- slow-check-interleaving/             (LEAF)  fast checks before delayed fetch
      +-- done-payload-includes-counts/        (LEAF)  done.healthy + checks_total/failed
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `progress-writer/emits-ordered-sse-frames` | `progress.Writer` emits 4 SSE `data:` frames in order |
| 2 | `progress-writer/content-type-event-stream` | Response `Content-Type` contains `text/event-stream` |
| 3 | `doctor-stream/server-checks-emit-in-order` | ≥5 server `progress` events; ids unique and ordered |
| 4 | `doctor-stream/slow-check-interleaving` | `config_load`/`upstream_proxy` arrive before delayed `upstream_fetch` |
| 5 | `doctor-stream/done-payload-includes-counts` | Terminal `done` has `healthy`, `try_url`, `checks_total`, `checks_failed` |

## Parameter Coverage

| Parameter | Significance | Values covered |
|-----------|--------------|----------------|
| `Target` | highest | `progress-writer`, `doctor-stream` |
| `SimulateXray` | high | true (doctor leaves) |
| `StubNetworkChecks` | high | true — skip/stub upstream fetch and public WS |
| `UpstreamFetchDelayMs` | medium | 0 (default), 200 (`slow-check-interleaving`) |
| `TryURL` | low | `https://example.com` |

## How to Run

```sh
doctest vet ./server/proxy/wsproxy/tests/streaming-doctor
doctest test ./server/proxy/wsproxy/tests/streaming-doctor/...
```

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

const (
	TargetProgressWriter = "progress-writer"
	TargetDoctorStream   = "doctor-stream"
)

type SSEEvent struct {
	Type    string
	Raw     json.RawMessage
	Decoded map[string]any
}

type Request struct {
	Target               string
	TryURL               string
	SimulateXray         bool
	StubNetworkChecks    bool
	UpstreamFetchDelayMs int
}

type Response struct {
	ContentType          string
	Events               []SSEEvent
	ProgressIDs          []string
	ProgressTimestamps   map[string]time.Time
	DoneHealthy          *bool
	DoneTryURL           string
	DoneChecksTotal      int
	DoneChecksFailed     int
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.TryURL == "" {
		req.TryURL = "https://example.com"
	}
	if req.Target == "" {
		req.Target = TargetDoctorStream
	}

	resp := &Response{
		ProgressTimestamps: make(map[string]time.Time),
	}

	rec := httptest.NewRecorder()

	switch req.Target {
	case TargetProgressWriter:
		w := progress.NewWriter(rec)
		if w == nil {
			return nil, fmt.Errorf("progress.NewWriter returned nil (ResponseRecorder must implement http.Flusher)")
		}
		if err := w.EmitProgress(progress.Item{
			ID: "step_a", Layer: "test", Name: "first", Status: "ok", Detail: "a",
		}); err != nil {
			return nil, err
		}
		if err := w.EmitProgress(progress.Item{
			ID: "step_b", Layer: "test", Name: "second", Status: "ok", Detail: "b",
		}); err != nil {
			return nil, err
		}
		if err := w.EmitSection("Checks"); err != nil {
			return nil, err
		}
		if err := w.EmitDone(map[string]any{"healthy": true}); err != nil {
			return nil, err
		}

	case TargetDoctorStream:
		if req.StubNetworkChecks {
			wsproxy.SetTestStubNetworkChecks(true)
			t.Cleanup(func() { wsproxy.SetTestStubNetworkChecks(false) })
		}
		if req.UpstreamFetchDelayMs > 0 {
			wsproxy.SetTestUpstreamFetchDelay(time.Duration(req.UpstreamFetchDelayMs) * time.Millisecond)
			t.Cleanup(func() { wsproxy.SetTestUpstreamFetchDelay(0) })
		}

		tmpDir := t.TempDir()
		wsproxy.SetTestConfigDir(tmpDir)
		t.Cleanup(func() { wsproxy.SetTestConfigDir("") })

		cfg := &wsproxy.Config{
			UpstreamProxy: "http://proxy.internal:3128",
			ListenPort:    0,
			WSPath:        "/ws",
			UUID:          "00000000-0000-4000-8000-000000000001",
			Subdomain:     "ws",
			InstanceID:    "25b2a55939e4",
			AutoStart:     true,
		}

		var xraySrv *httptest.Server
		if req.SimulateXray {
			xraySrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == cfg.WSPath {
					http.Error(w, "Bad Request", http.StatusBadRequest)
					return
				}
				http.NotFound(w, r)
			}))
			defer xraySrv.Close()
			cfg.ListenPort = wsproxy.ExtractPortFromURL(xraySrv.URL)
		}

		if err := wsproxy.SaveTestConfig(cfg); err != nil {
			return nil, err
		}

		publicURL := "https://ws-25b2a55939e4.xhd2015.xyz"
		m := wsproxy.NewTestManager(publicURL, false)

		if err := m.DoctorStream(rec, req.TryURL); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown Target %q", req.Target)
	}

	resp.ContentType = rec.Header().Get("Content-Type")
	events, err := parseSSEBody(rec.Body.String())
	if err != nil {
		return nil, err
	}
	resp.Events = events

	for _, ev := range events {
		if ev.Type != "progress" {
			continue
		}
		id, _ := ev.Decoded["id"].(string)
		if id == "" {
			continue
		}
		resp.ProgressIDs = append(resp.ProgressIDs, id)
		resp.ProgressTimestamps[id] = time.Now()
	}

	for _, ev := range events {
		if ev.Type != "done" {
			continue
		}
		if h, ok := ev.Decoded["healthy"].(bool); ok {
			resp.DoneHealthy = &h
		}
		if u, ok := ev.Decoded["try_url"].(string); ok {
			resp.DoneTryURL = u
		}
		if n, ok := ev.Decoded["checks_total"].(float64); ok {
			resp.DoneChecksTotal = int(n)
		}
		if n, ok := ev.Decoded["checks_failed"].(float64); ok {
			resp.DoneChecksFailed = int(n)
		}
	}

	return resp, nil
}

func parseSSEBody(body string) ([]SSEEvent, error) {
	var events []SSEEvent
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var decoded map[string]any
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			return nil, fmt.Errorf("decode SSE frame: %w", err)
		}
		typ, _ := decoded["type"].(string)
		events = append(events, SSEEvent{
			Type:    typ,
			Raw:     json.RawMessage(payload),
			Decoded: decoded,
		})
	}
	return events, nil
}

func eventTypes(events []SSEEvent) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = ev.Type
	}
	return out
}

func serverProgressLayer(events []SSEEvent) []SSEEvent {
	var out []SSEEvent
	for _, ev := range events {
		if ev.Type != "progress" {
			continue
		}
		if layer, _ := ev.Decoded["layer"].(string); layer == "server" {
			out = append(out, ev)
		}
	}
	return out
}

func expectedServerDoctorCheckOrder() []string {
	return []string{
		"config_load",
		"upstream_proxy",
		"uuid",
		"xray_binary",
		"xray_process",
		"local_xray_health",
		"listen_port",
		"public_url",
		"quick_tunnel",
		"extension_tunnel",
		"tunnel_ingress",
		"public_ws_endpoint",
		"upstream_tcp",
		"upstream_fetch",
		"client_ready",
	}
}
```