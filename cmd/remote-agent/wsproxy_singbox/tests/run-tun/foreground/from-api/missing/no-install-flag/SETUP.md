# Scenario

**Feature**: `--no-install` fails fast when sing-box missing

```
# --no-install: never brew, immediate error
--no-install -> error (no BrewInstall)
```

## Steps

1. TTY with `--no-install` (flag applies regardless of TTY).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = true
	req.NoInstall = true
	return nil
}
```