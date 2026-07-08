# Scenario

**Feature**: transport reset on chunk 2 recovers after retry

```
# RoundTripper fails twice with connection reset; third try reaches server
transport fail x2 -> chunk[2] server POST x1 -> complete
```

## Preconditions

Inherited from `transient-recovery/SETUP.md` with transport injection.

## Steps

1. Set `TransportFailChunk=2`, `TransportFailCount=2`, disable HTTP failure injection.

## Context

Exercises full `UploadFile` → `uploadChunkWithRetry` → `http.Client.Do` transport error path.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TransportFailChunk = 2
	req.TransportFailCount = 2
	req.TransientFails = 0
	req.FlakyChunkIndex = -1
	req.PermanentStatus = 0
	req.AlwaysFailChunk = -1
	return nil
}
```