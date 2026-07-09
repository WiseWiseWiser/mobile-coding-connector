# Scenario

**Feature**: saved config file has mode 0600

```
Save(path, cfg) -> os.Stat mode == 0600
```

## Preconditions

Config contains at least one domain (token present — file must not be world-readable).

## Steps

1. Minimal ConfigJSON; no special mutation.

## Context

REQUIREMENT leaf: `save/file-mode-0600` (mode on write: `0600`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://example.com",
  "domains": [
    {"server": "https://example.com", "token": "secret-mode"}
  ]
}`
	return nil
}
```
