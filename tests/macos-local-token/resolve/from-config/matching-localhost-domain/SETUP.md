# Scenario

**Feature**: domain matching localhost:23712 wins when default points elsewhere

```
# default is remote; local domain holds the token
domains[localhost:23712].token=local-host-token
  -> ResolveLocalServerToken -> local-host-token, source=config
```

## Preconditions

A domain entry matches `http://localhost:23712` after normalize; `default` points
at a different server (so success is via local match, not merely default lookup).

## Steps

1. Write config with default `https://other.example.com` and a localhost domain token.
2. Do not write credentials.

## Context

REQUIREMENT leaf: scenario 5 — matching localhost domain token (not default).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://other.example.com",
  "domains": [
    {"server": "https://other.example.com", "token": "other-token"},
    {"server": "http://localhost:23712", "token": "local-host-token"}
  ]
}
`
	req.CredentialsPresent = false
	return nil
}
```
