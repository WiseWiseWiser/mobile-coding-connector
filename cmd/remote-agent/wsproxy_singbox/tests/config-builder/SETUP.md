# Scenario

**Feature**: pure `BuildSingBoxTunConfig` rendering

```
# VMess params in -> sing-box JSON out (no CLI, no network)
BuildSingBoxTunConfig(VMessParams) -> JSON (inbounds/outbounds/route/dns)
```

## Steps

1. Set `Request.Op = OpBuildConfig`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpBuildConfig
	return nil
}
```