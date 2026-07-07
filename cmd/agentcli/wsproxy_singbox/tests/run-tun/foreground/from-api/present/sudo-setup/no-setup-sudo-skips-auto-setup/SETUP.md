# Scenario

**Feature**: --no-setup-sudo skips EnsureSudoSetup before sing-box launch

```
# NoSetupSudo=true -> no EnsureSudoSetup -> RunSingBox (sudo cache only)
```

## Preconditions

- User opts out of auto sudoers setup.
- sing-box on PATH, non-root with TTY.

## Steps

1. Set `NoSetupSudo=true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NoSetupSudo = true
	return nil
}
```