# Scenario

**Feature**: streamcmd.Run prints SSE events incrementally

```
# mock SSE -> client.Stream -> effective printer -> stdout
streamcmd.Run(Print, Printer) -> captured stdout/stderr
```

## Preconditions

- `streamcmd` package exists with `Spec`, `PrintFlags`, `Printer`, `Run`.
- Mock SSE server reachable via `client.New(mockURL)`.

## Steps

1. Child `Setup` sets `Print`, `Printer`, and `MockEvents`.
2. Root `Run` redirects stdout/stderr, calls `streamcmd.Run`, restores fds.

## Context

CLI framework unit tests — no real `remote-agent` subprocess.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.MockEvents) == 0 {
		req.MockEvents = defaultStreamcmdEvents()
	}
	return nil
}
```
