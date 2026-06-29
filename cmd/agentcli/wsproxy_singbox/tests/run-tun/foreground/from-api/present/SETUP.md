# Scenario

**Feature**: run-tun when sing-box is already on PATH

```
# LookPath finds sing-box — skip brew install path
LookPath("sing-box") -> found -> privilege check -> RunSingBox
```

## Steps

1. Set `SingBoxOnPath = true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SingBoxOnPath = true
	return nil
}
```