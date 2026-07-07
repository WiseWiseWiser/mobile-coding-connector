# Scenario

**Feature**: default run-tun calls EnsureSudoSetup before sudo sing-box

```
# NoSetupSudo=false, not yet installed -> EnsureSudoSetup called -> RunSingBox
```

## Preconditions

- Auto-setup enabled (default).
- Sudo rule not yet installed.

## Steps

1. Leave `NoSetupSudo=false`.
2. Leave `SudoSetupInstalled=false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NoSetupSudo = false
	req.SudoSetupInstalled = false
	return nil
}
```