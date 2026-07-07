# Scenario

**Feature**: EnsureSudoSetup noop when NOPASSWD rule already installed

```
# SudoSetupInstalled=true -> EnsureSudoSetup called but skips install -> RunSingBox
```

## Preconditions

- Auto-setup enabled.
- Mock reports rule already installed.

## Steps

1. Set `SudoSetupInstalled=true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NoSetupSudo = false
	req.SudoSetupInstalled = true
	return nil
}
```