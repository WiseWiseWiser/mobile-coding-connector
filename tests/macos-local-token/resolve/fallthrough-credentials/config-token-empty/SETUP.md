# Scenario

**Feature**: whitespace-only config token falls through to credentials

```
domains[localhost].token="   \t  " + credentials "cred-after-empty-cfg"
  -> token=cred-after-empty-cfg, source=credentials
```

## Preconditions

Config parses but domain token is empty after trim; credentials has a usable line.

## Steps

1. Write valid config with whitespace-only token on matching localhost domain.
2. Write credentials with `cred-after-empty-cfg`.

## Context

REQUIREMENT leaf: scenario 3.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = true
	req.ConfigJSON = `{
  "default": "http://localhost:23712",
  "domains": [
    {"server": "http://localhost:23712", "token": "   \t  "}
  ]
}
`
	req.CredentialsText = "cred-after-empty-cfg\n"
	return nil
}
```
