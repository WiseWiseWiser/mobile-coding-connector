# Scenario

**Feature**: server binary leaves no opencode serve orphans

```
ai-critic-server -> POST launch grok -> stop or SIGTERM -> zero session port listeners
```

## Preconditions

- Server binary built from module root once per leaf.
- Fake opencode on PATH (`UseFakeOpenCode=true`) for deterministic fast runs.
- `lib.CleanupOpencodeServe` invoked from `stopServer` (implementer wires production + harness).
- Test auth token `lib.TestPassword`.

## Steps

1. Child `Setup` sets `Request.Scenario`.
2. Root `Run` starts server, launches grok session, runs stop or shutdown path.
3. Leaf `Assert` verifies port and registry state.

## Context

Classic TDD: RED until registry persistence, CleanupAll on shutdown, and harness fix land.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseFakeOpenCode = true
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 45
	}
	return nil
}
```
