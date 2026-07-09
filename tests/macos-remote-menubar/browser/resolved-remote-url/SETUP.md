# Scenario

**Feature**: Open in Browser uses resolved remote server, not localhost

```
OpenBrowserURL(ResolvedEndpoint{Server: https://remote.example, OK: true})
  -> https://remote.example
```

## Preconditions

Endpoint resolved to a remote HTTPS base URL.

## Steps

1. Set browser endpoint fields for a remote server.

## Context

REQUIREMENT leaf: `browser/resolved-remote-url`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.BrowserServer = "https://remote.example"
	req.BrowserToken = "tok"
	req.BrowserOK = true
	return nil
}
```
