# Scenario

**Feature**: empty local domain token falls through to default domain within config

```
localhost domain token="" + default domain token=from-default-after-local-empty
  -> ResolveLocalServerToken -> from-default-after-local-empty, source=config
```

## Preconditions

Local loopback domain is present but token empty after trim; default matches a
different domain with a non-empty token. Config step tries local first, then
default, before credentials.

## Steps

1. Write config with empty localhost token and a usable default domain token.
2. Do not write credentials.

## Context

REQUIREMENT: prefer local match, else default domain (still source=config).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Local loopback domain present but empty; default points at a *remote*
	// domain so success must come from the default-domain branch (not another
	// loopback match).
	req.ConfigJSON = `{
  "default": "https://remote.example.com",
  "domains": [
    {"server": "http://localhost:23712", "token": ""},
    {"server": "https://remote.example.com", "token": "from-default-after-local-empty"}
  ]
}
`
	req.CredentialsPresent = false
	return nil
}
```
