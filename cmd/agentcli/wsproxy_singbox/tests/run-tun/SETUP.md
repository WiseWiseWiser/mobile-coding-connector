# Scenario

**Feature**: `ws-proxy sing-box run-tun` orchestrates sing-box TUN mode

```
# resolve config, ensure binary, run foreground or detached
CLI run-tun -> config resolve -> LookPath -> [brew?] -> RunSingBox | StartDetached
```

## Steps

1. Set `Request.Op = OpRunTun`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRunTun
	return nil
}
```