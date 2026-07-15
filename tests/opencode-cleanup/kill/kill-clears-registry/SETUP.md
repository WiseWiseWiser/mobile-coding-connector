# Scenario

**Feature**: CleanupOpencodeServe kills children and clears registry

```
fixture registry + fake opencode -> CleanupOpencodeServe -> empty registry
```

## Preconditions

- Registry populated with fake opencode child PID/port.

## Steps

1. `Op = OpCleanup`.

## Context

Production shutdown and doctest `stopServer` call this after killing PIDs.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpCleanup
	return nil
}
```
