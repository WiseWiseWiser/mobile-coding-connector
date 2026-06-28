# Scenario

**Feature**: run-tun when sing-box is not on PATH

```
# LookPath misses sing-box — install policy decides outcome
LookPath("sing-box") -> miss -> [confirm/brew | error]
```

## Steps

1. Set `SingBoxOnPath = false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SingBoxOnPath = false
	return nil
}
```