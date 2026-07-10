# Scenario

**Feature**: domain matching 127.0.0.1:23712 resolves as config token

```
domains[http://127.0.0.1:23712].token=local-127-token
  -> ResolveLocalServerToken -> local-127-token, source=config
```

## Preconditions

Local match targets include both `http://localhost:23712` and
`http://127.0.0.1:23712` after normalize.

## Steps

1. Write config with a 127.0.0.1 domain token; default may be empty or matching.
2. Do not write credentials.

## Context

REQUIREMENT leaf: loopback variant of scenario 5 (127.0.0.1 form).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "",
  "domains": [
    {"server": "http://127.0.0.1:23712", "token": "local-127-token"}
  ]
}
`
	req.CredentialsPresent = false
	return nil
}
```
