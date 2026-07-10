# Scenario

**Feature**: config token wins when credentials also present

```
config token=cfg-wins-token + credentials first line=cred-should-not-win
  -> ResolveLocalServerToken -> cfg-wins-token, source=config
```

## Preconditions

Both sources have non-empty tokens; locked order never skips a usable config token.

## Steps

1. Write config with default localhost domain token `cfg-wins-token`.
2. Write credentials with `cred-should-not-win`.

## Context

REQUIREMENT: fall-through only when config fails; not when both succeed.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "http://localhost:23712",
  "domains": [
    {"server": "http://localhost:23712", "token": "cfg-wins-token"}
  ]
}
`
	req.CredentialsPresent = true
	req.CredentialsText = "cred-should-not-win\n"
	return nil
}
```
