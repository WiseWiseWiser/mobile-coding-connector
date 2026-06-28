# Scenario

**Feature**: non-root foreground run invokes sudo sing-box

```
# euid≠0 + TTY: sudo sing-box run -c <config>
Geteuid != 0 -> RunSingBox(sudo=true)
```

## Steps

1. Non-root EUID with TTY available.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.EUID = euidPtr(1000)
	req.IsTTY = true
	return nil
}
```