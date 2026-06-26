# Scenario

**Feature**: Client.Stream error termination

```
# fatal error frame or truncated stream surfaces as Go error
SSE error / truncated -> Stream returns err
```

## Preconditions

`MockEvents` configured by leaf to omit `done` or include `error`.

## Steps

Child leaves set `MockEvents` for their failure mode.

## Context

Error-path coverage for the transport layer.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Path == "" {
		req.Path = "/mock/stream"
	}
	return nil
}
```
