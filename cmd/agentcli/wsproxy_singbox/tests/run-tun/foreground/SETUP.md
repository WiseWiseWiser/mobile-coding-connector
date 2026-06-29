# Scenario

**Feature**: run-tun foreground mode streams sing-box logs

```
# default: no --detach, RunSingBox blocks until context cancelled
run-tun (foreground) -> RunSingBox (blocking)
```

## Steps

1. Set `Detach = false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Detach = false
	return nil
}
```