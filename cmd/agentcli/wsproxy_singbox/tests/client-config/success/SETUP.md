# Scenario

**Feature**: client-config succeeds when ws-proxy returns VMess params

```
# API ready: FetchVMess returns host/port/path for config builder
FetchVMess -> VMessParams -> BuildSingBoxTunConfig
```

## Preconditions

- `FetchVMess` returns default mock VMess (no error).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FetchVMessErr = nil
	return nil
}
```