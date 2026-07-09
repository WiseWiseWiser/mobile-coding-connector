# Scenario

**Feature**: missing config file is not_configured

```
Load(missing path) -> nil config -> Resolve -> state=not_configured, no endpoint
```

## Preconditions

Config file does not exist at `ConfigPath`.

## Steps

1. Create empty temp dir; set path to non-existent config file.
2. `UseLoad=true` so Run loads then resolves.

## Context

REQUIREMENT leaf: `resolve/missing-file`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir, err := os.MkdirTemp("", "macos-remote-menubar-missing-*")
	if err != nil {
		return err
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	req.UseLoad = true
	req.ConfigPath = filepath.Join(dir, "remote-agent-config.json")
	// Do not create the file.
	return nil
}
```
