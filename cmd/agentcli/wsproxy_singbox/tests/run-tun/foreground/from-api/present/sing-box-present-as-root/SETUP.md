# Scenario

**Feature**: root foreground run invokes sing-box directly

```
# euid=0: sing-box run without sudo
Geteuid == 0 -> RunSingBox(sudo=false)
```

## Steps

1. Set `EUID = 0` and `IsTTY = true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.EUID = euidPtr(0)
	req.IsTTY = true
	return nil
}
```