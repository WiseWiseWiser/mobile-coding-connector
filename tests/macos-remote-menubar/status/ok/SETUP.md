# Scenario

**Feature**: ok status shows Connected to {server} without token

```
FormatStatus(ok, "https://example.com") -> "Connected to https://example.com"
```

## Preconditions

Resolved remote server; network/auth probe succeeded.

## Steps

1. Set state `ok` and server URL; pass a sentinel token only to detect leakage.

## Context

REQUIREMENT leaf: `status/ok`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConnectionState = "ok"
	req.StatusServer = "https://example.com"
	req.Token = "super-secret-token-xyz"
	return nil
}
```
