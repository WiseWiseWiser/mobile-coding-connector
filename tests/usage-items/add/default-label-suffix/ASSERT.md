## Expected

1. The first item is labeled `CommandCode` with title `CommandCode 5%`.
2. The second item is labeled `CommandCode cc-v2` with title `CommandCode cc-v2 5%`.
3. No warnings: both registrations validated against their provider.

## Errors

- Leaving both items labeled `CommandCode`, or suffixing with the kind instead of the id.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Added) != 2 {
		t.Fatalf("added = %d, want 2", len(resp.Added))
	}
	if len(resp.Warnings) != 0 {
		t.Fatalf("warnings = %v", resp.Warnings)
	}
	first, second := resp.Added[0], resp.Added[1]
	if first.Label != "CommandCode" || first.Title != "CommandCode 5%" {
		t.Fatalf("first = %q / %q", first.Label, first.Title)
	}
	if second.Label != "CommandCode cc-v2" || second.Title != "CommandCode cc-v2 5%" {
		t.Fatalf("second = %q / %q", second.Label, second.Title)
	}
	if !first.Enabled || !second.Enabled {
		t.Fatal("new items must be enabled by default")
	}
}
```
