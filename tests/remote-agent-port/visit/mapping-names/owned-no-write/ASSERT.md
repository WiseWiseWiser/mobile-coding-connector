## Expected

1. Start succeeds with owned provider.
2. MappingNamesAfter equals seed (still only 9999 → keep.example.com).
3. No new key for the visited port.

```go
import (
	"strconv"
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
	if resp.Session == nil {
		t.Fatal("expected session")
	}
	p := resp.Session.Provider
	if p != "cloudflare_owned" && p != "owned" {
		t.Fatalf("provider=%q want owned", p)
	}
	if resp.MappingNamesAfter["9999"] != "keep.example.com" {
		t.Fatalf("seed entry lost: %+v", resp.MappingNamesAfter)
	}
	portKey := strconv.Itoa(defaultTestPort)
	if _, ok := resp.MappingNamesAfter[portKey]; ok {
		t.Fatalf("owned ad-hoc must not write mapping-names for port %s; got %+v", portKey, resp.MappingNamesAfter)
	}
	if len(resp.MappingNamesAfter) != 1 {
		t.Fatalf("mapping-names should be unchanged; got %+v", resp.MappingNamesAfter)
	}
}
```
