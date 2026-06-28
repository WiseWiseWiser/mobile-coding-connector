# Scenario

**Feature**: LAN/private CIDRs route to direct outbound

```
# LANBypass: 10/8, 172.16/12, 192.168/16 -> route rules outbound direct
BuildSingBoxTunConfig -> route.rules (ip_cidr -> direct)
```

## Steps

1. Use default VMess params.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.VMessFixture = ""
	return nil
}
```