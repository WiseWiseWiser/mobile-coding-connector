# Scenario

**Feature**: --set-config is standalone (mutually exclusive with dry-run)

```
# set-config combined with --dry-run -> CLI error
remote-agent machine backup --set-config --exclude .cache --dry-run -> non-zero exit
```

## Preconditions

Default `serverHome` fixtures.

## Steps

1. `SetConfig=true`, `ExcludePaths=[".cache"]`.
2. Args: `machine backup --dry-run`.

## Context

Validation leaf: set-config cannot combine with backup operation flags.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SetConfig = true
	req.ExcludePaths = []string{".cache"}
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```