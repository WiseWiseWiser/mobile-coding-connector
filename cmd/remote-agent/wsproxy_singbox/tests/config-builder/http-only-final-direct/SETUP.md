# Scenario

**Feature**: http-only config uses final direct and catch-all for default blacklist

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpBuildHttpOnlyConfig
	req.HttpOnlySocksPort = 11080
	req.HttpOnlyDNSHijack = true
	return nil
}
```