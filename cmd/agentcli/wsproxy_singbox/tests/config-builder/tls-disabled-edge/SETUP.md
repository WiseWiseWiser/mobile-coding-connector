# Scenario

**Feature**: VMess tls field "none" disables TLS on outbound

```
# tls:"none" edge case -> tls.enabled false
VMessParams(tls=none) -> outbound.tls.enabled=false
```

## Steps

1. Load `testdata/vmess-no-tls.json`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.VMessFixture = "testdata/vmess-no-tls.json"
	return nil
}
```