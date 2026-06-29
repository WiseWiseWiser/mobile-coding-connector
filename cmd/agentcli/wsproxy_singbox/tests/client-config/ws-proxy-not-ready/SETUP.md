# Scenario

**Feature**: client-config errors when ws-proxy is not client-ready

```
# FetchVMess fails with NOT_RUNNING — no config emitted
FetchVMess -> error (ws-proxy not running)
```

## Steps

1. Configure `FetchVMessErr` to simulate API NOT_RUNNING response.

```go
import (
	"errors"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.FetchVMessErr = errors.New("ws-proxy is not running")
	return nil
}
```