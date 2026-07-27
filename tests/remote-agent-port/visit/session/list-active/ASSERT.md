## Expected

1. Sessions length 1.
2. Fields: ID, LocalPort, PublicURL, Provider, Status active-ish.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr != "" {
		t.Fatalf("StartErr: %s", resp.StartErr)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("want 1 session, got %+v", resp.Sessions)
	}
	s := resp.Sessions[0]
	if s.ID == "" {
		t.Fatal("session ID empty")
	}
	if s.LocalPort != defaultTestPort {
		t.Fatalf("LocalPort=%d", s.LocalPort)
	}
	if strings.TrimSpace(s.PublicURL) == "" {
		t.Fatal("PublicURL empty")
	}
	if s.Provider == "" {
		t.Fatal("Provider empty")
	}
}
```
