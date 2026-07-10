# Scenario

**Feature**: non-empty default domain token resolves from config

```
default + domain token "cfg-default-token"
  -> ResolveLocalServerToken -> token=cfg-default-token, source=config
```

## Preconditions

`default` matches a domain with non-empty token (local loopback URL).

## Steps

1. Write config with default `http://localhost:23712` and matching domain token.
2. Do not write credentials.

## Context

REQUIREMENT leaf: scenario 1 — config default domain token.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "http://localhost:23712",
  "domains": [
    {"server": "http://localhost:23712", "token": "cfg-default-token"}
  ]
}
`
	req.CredentialsPresent = false
	return nil
}
```
