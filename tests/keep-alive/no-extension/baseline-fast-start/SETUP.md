# Scenario

**Feature**: core-only startup is fast without extension/tunnel work

```
skip extension hook -> core_ready within 3s -> daemon stable
```

## Preconditions

`SkipExtensionStartup=true`, no extension opencode trigger.

## Steps

1. `ObserveSecs=8`.

## Context

Baseline latency budget for core bootstrap.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ObserveSecs = 8
	return nil
}
```