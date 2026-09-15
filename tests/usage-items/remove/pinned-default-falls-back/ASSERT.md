## Expected

1. The removed id is reported.
2. The default becomes the first enabled item, `grok`, and the change is reported as a warning.
3. The registry file no longer contains `cc-v1`.

## Errors

- Leaving the default pointing at the removed id, or reassigning it silently.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Removed != "cc-v1" || resp.Default != "grok" {
		t.Fatalf("result = removed %q, default %q", resp.Removed, resp.Default)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0] != "cc-v1 was the menu bar default; default is now grok" {
		t.Fatalf("warnings = %v", resp.Warnings)
	}
	reg := readRegistry(t, resp.Registry)
	if len(reg.Items) != 3 {
		t.Fatalf("persisted items = %d, want 3", len(reg.Items))
	}
	for _, item := range reg.Items {
		if item.ID == "cc-v1" {
			t.Fatal("cc-v1 must be gone from the registry file")
		}
	}
}
```
