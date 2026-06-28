# Scenario

**Feature**: run-tun builds config from live VMess API

```
# no --config: FetchVMess + BuildSingBoxTunConfig before sing-box start
FetchVMess -> BuildSingBoxTunConfig -> sing-box run
```

## Preconditions

- `ConfigFile` is empty (fetch from API).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigFile = ""
	return nil
}
```