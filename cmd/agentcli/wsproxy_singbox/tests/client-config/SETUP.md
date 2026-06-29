# Scenario

**Feature**: `ws-proxy sing-box client-config` emits TUN-ready JSON

```
# fetch live VMess params, render sing-box config, write stdout or --output
CLI client-config -> FetchVMess -> BuildSingBoxTunConfig -> stdout | --output FILE
```

## Steps

1. Set `Request.Op = OpClientConfig`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpClientConfig
	return nil
}
```