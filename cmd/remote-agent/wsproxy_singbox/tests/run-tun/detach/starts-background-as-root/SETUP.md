# Scenario

**Feature**: detached root start runs without sudo

```
# euid=0 + --detach: StartDetached(useSudo=false)
Geteuid == 0 -> StartDetached(useSudo=false)
```

## Steps

1. Root EUID.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.EUID = euidPtr(0)
	return nil
}
```