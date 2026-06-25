## Preconditions

xray healthy, publicURL set, tunnel ingress mapping intentionally absent.

## Steps

1. Enable `SimulateXray`.
2. Do not add tunnel mapping.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SimulateXray = true
	req.AddTunnelMapping = false
	return nil
}
```