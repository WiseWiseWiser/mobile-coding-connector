# Scenario

**Feature**: TUN inbound uses auto_route and strict_route defaults

```
# SingBoxTunOptions defaults: auto_route=true, strict_route=true
BuildSingBoxTunConfig -> inbounds[type=tun]
```

## Steps

1. Use default mock VMess via empty fixture (root default).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.VMessFixture = ""
	return nil
}
```