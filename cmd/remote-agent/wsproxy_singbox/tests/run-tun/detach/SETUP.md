# Scenario

**Feature**: run-tun `--detach` starts sing-box in background

```
# --detach: StartDetached, print PID + paths, parent exits
StartDetached -> stdout (PID, config path, log path)
```

## Steps

1. Set `Detach = true` and `SingBoxOnPath = true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Detach = true
	req.SingBoxOnPath = true
	req.IsTTY = true
	return nil
}
```