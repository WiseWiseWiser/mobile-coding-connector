# Client Streaming Doctests

Unit tests for `client.Client.Stream` — the unified SSE consumer that decodes
typed progress envelopes and legacy `log` events.

# DSN (Domain Specific Notion)

The client streaming harness models the transport layer between ai-critic server
SSE endpoints and CLI consumers.

**Participants**

- **Client.Stream** — sets `Accept: text/event-stream`, scans `data:` lines,
  decodes JSON into `StreamEvent`, requires terminal `done` or `error`.
- **Mock SSE server** — `httptest.Server` returns canned event sequences.
- **StreamResult** — captures the terminal `done` payload map.

**Behaviors**

- Progress events are delivered to the caller in wire order.
- Legacy `log` events decode as `StreamEvent{Type: log, Message: ...}`.
- `error` events terminate the stream with a Go error.
- Streams without `done` or `error` return an error.

## Version

0.0.2

## Decision Tree

```
[client.Stream]
 |
 +-- happy-path/
 |    |
 |    +-- consumes-progress-and-done/        (LEAF)  3 progress + done
 |    +-- decodes-log-events/                 (LEAF)  legacy log type
 |    +-- preserves-event-order/              (LEAF)  callback order matches wire
 |
 +-- error-path/
      |
      +-- error-event-terminates/             (LEAF)  type=error → err
      +-- missing-done-fails/                 (LEAF)  stream without done
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `happy-path/consumes-progress-and-done` | 3 progress events then done; `StreamResult.Done` populated |
| 2 | `happy-path/decodes-log-events` | `log` events decode with `Message` field |
| 3 | `happy-path/preserves-event-order` | Callback receives events in SSE order |
| 4 | `error-path/error-event-terminates` | `error` frame returns non-nil error |
| 5 | `error-path/missing-done-fails` | Stream ending without `done`/`error` fails |

## How to Run

```sh
doctest vet ./client/tests/streaming
doctest test ./client/tests/streaming/...
```

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

type Request struct {
	MockEvents []map[string]any
	Method     string
	Path       string
}

type Response struct {
	Events     []client.StreamEvent
	Done       map[string]any
	StreamErr  string
	CallOrder  []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Method == "" {
		req.Method = http.MethodGet
	}
	if req.Path == "" {
		req.Path = "/mock/stream"
	}
	if len(req.MockEvents) == 0 {
		req.MockEvents = defaultProgressDoneSequence()
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := newSSEWriter(w)
		if sw == nil {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		for _, ev := range req.MockEvents {
			sendSSE(sw, ev)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")

	result, err := c.Stream(req.Method, req.Path, nil)
	if result != nil && len(result.Events) > 0 {
		resp.Events = result.Events
		for _, ev := range result.Events {
			resp.CallOrder = append(resp.CallOrder, ev.Type+":"+ev.ID)
		}
	}

	if err != nil {
		resp.StreamErr = err.Error()
		return resp, nil
	}
	if result != nil {
		resp.Done = result.Done
	}
	return resp, nil
}

func defaultProgressDoneSequence() []map[string]any {
	return []map[string]any{
		{"type": "progress", "id": "a", "layer": "server", "name": "first", "status": "ok"},
		{"type": "progress", "id": "b", "layer": "server", "name": "second", "status": "ok"},
		{"type": "progress", "id": "c", "layer": "server", "name": "third", "status": "ok"},
		{"type": "done", "healthy": true, "checks_total": 3, "checks_failed": 0},
	}
}

type mockSSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter) *mockSSEWriter {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	return &mockSSEWriter{w: w, flusher: f}
}

func sendSSE(sw *mockSSEWriter, v map[string]any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(sw.w, "data: %s\n\n", data)
	sw.flusher.Flush()
}
```