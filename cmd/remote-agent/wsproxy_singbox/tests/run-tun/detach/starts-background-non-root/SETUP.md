# Scenario

**Feature**: detached non-root start uses sudo in background

```
# euid≠0 + --detach: StartDetached(useSudo=true)
Geteuid != 0 -> StartDetached(useSudo=true) -> PID 4242
```

## Steps

1. Non-root EUID; default DetachPID 4242.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.EUID = euidPtr(1000)
	return nil
}
```