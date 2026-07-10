# Scenario

**Feature**: local loopback domain token preferred over default domain token

```
default domain token=default-domain-token
localhost domain token=prefer-local-token
  -> ResolveLocalServerToken -> prefer-local-token, source=config
```

## Preconditions

Both default-matched domain and local-loopback domain have non-empty tokens;
locked order prefers local match first.

## Steps

1. Write config with both a remote default domain and a localhost domain.
2. Do not write credentials.

## Context

REQUIREMENT: "Prefer domain matching local server … else default domain".

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://remote.example.com",
  "domains": [
    {"server": "https://remote.example.com", "token": "default-domain-token"},
    {"server": "http://localhost:23712/", "token": "prefer-local-token"}
  ]
}
`
	req.CredentialsPresent = false
	return nil
}
```
