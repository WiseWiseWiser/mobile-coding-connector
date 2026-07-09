# Scenario

**Feature**: resolve remote-agent-config to endpoint + config-level state

```
Config (domains, default) -> remoteconfig.Resolve -> (ResolvedEndpoint, ConnectionState)
# states: not_configured | no_default | ok (config usable; network probed separately)
```

## Preconditions

`remoteconfig.Resolve(cfg *Config) (ResolvedEndpoint, ConnectionState)` implements
default-match, single-domain fallback, multi-domain `no_default`, and server
normalize (trim space + trailing `/`).

## Steps

1. Set `Op=resolve`.
2. Leaf supplies `ConfigJSON` or `UseLoad` + `ConfigPath`.

## Context

REQUIREMENT group: `resolve/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "resolve"
	return nil
}
```
