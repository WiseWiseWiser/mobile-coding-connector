# Scenario

**Feature**: pure resolve of local server Bearer token from DataDir fixtures

```
# ResolveLocalServerToken reads only under opts.DataDir
DataDir/local-agent-config.json + DataDir/server-credentials
  -> localauth.ResolveLocalServerToken
  -> (token, source)
```

## Preconditions

`localauth.ResolveLocalServerToken` implements the locked order with fall-through
on missing/empty/invalid config and missing/empty credentials. Tests never use
real home; `Run` writes fixtures into a temp `DataDir`.

## Steps

1. Set `Op=resolve`.
2. Leaf sets `ConfigPresent` / `ConfigJSON` and `CredentialsPresent` / `CredentialsText`.

## Context

REQUIREMENT group: `resolve/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "resolve"
	return nil
}
```
