# Scenario

**Feature**: non-root non-TTY cannot invoke sudo sing-box

```
# euid≠0 + !TTY: privilege error before RunSingBox
Geteuid != 0, !IsTTY -> error (TTY or root required)
```

## Steps

1. Non-root without TTY.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.EUID = euidPtr(1000)
	req.IsTTY = false
	return nil
}
```