# streamcmd Doctests

Unit tests for `streamcmd.Run` — the CLI streaming framework with declarative
`Print` flags (A) and per-type `Printer` overrides (B).

# DSN (Domain Specific Notion)

The streamcmd harness models how remote-agent commands turn SSE events into
incremental terminal output.

**Participants**

- **streamcmd.Run** — resolves effective handlers from `Print` + `Printer`,
  calls `client.Stream`, flushes stdout after each print.
- **Mock SSE server** — returns canned `log` / `section` / `progress` / `done`.
- **bytes.Buffer stdout** — captures printed lines for assertion.
- **Printer overrides** — per-type `EventHandler` replacing A-path defaults.

**Behaviors**

- `Print: Logs | Sections | ProgressChecks` enables builtin formatters.
- Non-nil `Printer.Log` overrides log formatting only; other types keep A defaults.
- `After` hook runs after `done` before returning.
- Default error handler returns `errors.New(message)`.

## Version

0.0.2

## Decision Tree

```
[streamcmd.Run]
 |
 +-- print-flags-defaults/
 |    |
 |    +-- all-builtin-printers/               (LEAF)  A-path formatting
 |    +-- after-hook-on-done/                 (LEAF)  After runs post-done
 |
 +-- printer-overrides/
 |    |
 |    +-- log-override-wins/                 (LEAF)  B-path log override
 |
 +-- error-handling/
      |
      +-- default-error-returns/              (LEAF)  error frame → err
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `print-flags-defaults/all-builtin-printers` | Builtin `[ok]` / section / log formatting |
| 2 | `print-flags-defaults/after-hook-on-done` | `After` invoked with done payload |
| 3 | `printer-overrides/log-override-wins` | Custom `Printer.Log` replaces default |
| 4 | `error-handling/default-error-returns` | Default error handler returns message |

## How to Run

```sh
doctest vet ./cmd/remote-agent/streamcmd/tests
doctest test ./cmd/remote-agent/streamcmd/tests/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

type Request struct {
	MockEvents       []map[string]any
	Print            streamcmd.PrintFlags
	Printer          streamcmd.Printer
	After            func(done map[string]any) error
	OverrideLogToStderr bool
}

type Response struct {
	RunErr       string
	Stdout       string
	Stderr       string
	AfterCalled  bool
	AfterDone    map[string]any
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.MockEvents) == 0 {
		req.MockEvents = defaultStreamcmdEvents()
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := newStreamcmdSSEWriter(w)
		for _, ev := range req.MockEvents {
			sendStreamcmdSSE(sw, ev)
		}
	}))
	defer srv.Close()

	oldOut := os.Stdout
	oldErr := os.Stderr
	var outBuf, errBuf bytes.Buffer
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	doneCh := make(chan struct{})
	go func() {
		_, _ = io.Copy(&outBuf, rOut)
		close(doneCh)
	}()
	errDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&errBuf, rErr)
		close(errDone)
	}()

	c := client.New(srv.URL, "test-token")
	getClient := func() (*client.Client, error) { return c, nil }

	after := req.After
	if after == nil {
		after = func(map[string]any) error { return nil }
	}
	wrappedAfter := func(d map[string]any) error {
		resp.AfterCalled = true
		resp.AfterDone = d
		return after(d)
	}

	spec := streamcmd.Spec{
		Method: http.MethodGet,
		Path:   "/mock",
		Print:  req.Print,
		Printer: req.Printer,
		After:  wrappedAfter,
	}

	runErr := streamcmd.Run(getClient, spec)

	wOut.Close()
	wErr.Close()
	<-doneCh
	<-errDone
	os.Stdout = oldOut
	os.Stderr = oldErr

	resp.Stdout = outBuf.String()
	resp.Stderr = errBuf.String()
	if runErr != nil {
		resp.RunErr = runErr.Error()
	}
	return resp, nil
}

func defaultStreamcmdEvents() []map[string]any {
	return []map[string]any{
		{"type": "log", "message": "hello log"},
		{"type": "section", "message": "Server checks"},
		{"type": "progress", "id": "cfg", "layer": "server", "name": "configuration load", "status": "ok", "detail": "/data/ws-proxy.json"},
		{"type": "done", "healthy": true, "binary_path": "/tmp/x"},
	}
}

type streamcmdSSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newStreamcmdSSEWriter(w http.ResponseWriter) *streamcmdSSEWriter {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	return &streamcmdSSEWriter{w: w, flusher: f}
}

func sendStreamcmdSSE(sw *streamcmdSSEWriter, v map[string]any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(sw.w, "data: %s\n\n", data)
	sw.flusher.Flush()
}
```