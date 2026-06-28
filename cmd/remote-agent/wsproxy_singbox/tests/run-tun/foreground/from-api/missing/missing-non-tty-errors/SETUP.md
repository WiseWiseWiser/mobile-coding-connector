# Scenario

**Feature**: missing sing-box on non-TTY fails with install hint

```
# !TTY: cannot prompt — error sing-box not installed
!IsTTY -> error (brew install sing-box hint)
```

## Steps

1. Set `IsTTY = false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = false
	return nil
}
```