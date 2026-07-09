# Scenario

**Feature**: unauthorized status says token rejected + Configure…

```
FormatStatus(unauthorized, "https://example.com") -> token rejected + Configure…
```

## Preconditions

Server reachable but auth check rejected the token.

## Steps

1. Set `ConnectionState=unauthorized` and server; sentinel token for leak check.

## Context

REQUIREMENT leaf: `status/unauthorized`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConnectionState = "unauthorized"
	req.StatusServer = "https://example.com"
	req.Token = "bad-token-value"
	return nil
}
```
