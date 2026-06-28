# Scenario

**Feature**: `--yes` skips install confirm on TTY

```
# TTY + --yes: BrewInstall without Confirm
--yes -> BrewInstall (no Confirm prompt)
```

## Steps

1. TTY with `--yes`; sing-box still missing initially.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = true
	req.Yes = true
	req.EUID = euidPtr(1000)
	return nil
}
```