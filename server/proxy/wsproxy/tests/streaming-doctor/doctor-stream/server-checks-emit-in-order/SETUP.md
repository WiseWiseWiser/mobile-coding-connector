# Scenario

**Feature**: server doctor checks stream in legacy order

```
# each add() in serverDoctorChecks becomes an immediate progress event
serverDoctorChecks -> emit(progress) × N -> done
```

## Preconditions

Fake xray alive, tunnel mapping absent, network checks stubbed.

## Steps

Use default `UpstreamFetchDelayMs` (0).

## Context

Requirement scenario `server-checks-emit-in-order`. Proves checks are not
batched only into the terminal `done` frame.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UpstreamFetchDelayMs = 0
	return nil
}
```
