## Expected

1. No error.
2. `IsLoopback` true.
3. `ListenAddr` non-empty and contains a port (not `:0` unbound).

## Errors

- Bound to non-loopback interface.
- Empty Addr.

```go
import (
	"net"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ListenAddr == "" {
		t.Fatal("ListenAddr empty")
	}
	if !resp.IsLoopback {
		t.Fatalf("IsLoopback=false for Addr %q", resp.ListenAddr)
	}
	host, port, err := net.SplitHostPort(resp.ListenAddr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", resp.ListenAddr, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("host %q is not loopback", host)
	}
	if port == "" || port == "0" {
		t.Fatalf("port %q not an assigned listen port", port)
	}
	if strings.HasPrefix(host, "0.0.0.0") || host == "::" {
		t.Fatalf("must not bind all interfaces: %q", resp.ListenAddr)
	}
}
```
