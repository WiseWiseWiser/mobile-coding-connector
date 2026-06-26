# Scenario

**Feature**: ws-proxy doctor streams server checks over SSE

```
# DoctorStream runs checks and emits progress frames immediately
Manager.DoctorStream -> progress.Writer -> SSE data: frames

# doctest captures ResponseRecorder body and parses event order
httptest.ResponseRecorder <- DoctorStream / progress.Writer
```

## Preconditions

- `server/streaming/progress` package provides `Writer` with `EmitProgress`,
  `EmitSection`, `EmitDone`.
- `Manager.DoctorStream` exists and uses the same check sequence as `Doctor()`.
- Test hooks `SetTestStubNetworkChecks` and `SetTestUpstreamFetchDelay` stub
  slow/external network checks.

## Steps

1. Child `Setup` sets `Request.Target` and scenario-specific fields.
2. Root `Run` executes either `progress.Writer` directly or `Manager.DoctorStream`.
3. `Run` parses SSE frames into `Response.Events` and derived progress metadata.

## Context

These tests are fast, isolated server unit tests. They do not start
`ai-critic-server` or `remote-agent`; integration coverage lives under
`tests/streaming/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.TryURL == "" {
		req.TryURL = "https://example.com"
	}
	return nil
}
```
