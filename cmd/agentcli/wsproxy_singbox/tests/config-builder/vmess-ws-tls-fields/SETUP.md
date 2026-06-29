# Scenario

**Feature**: VMess WS+TLS outbound fields match server params

```
# golden outbound: server, port, transport.path, tls.server_name
VMessParams(host,port,path,tls) -> outbound vmess fields
```

## Steps

1. Load `testdata/vmess.json` fixture.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.VMessFixture = "testdata/vmess.json"
	return nil
}
```