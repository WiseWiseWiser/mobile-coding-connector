## Expected

1. `EmptyLabel` is exactly `No recent backups`.

## Errors

- Blank row, different wording, or reusing Terminals/Services empty labels.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "No recent backups"
	if resp.EmptyLabel != want {
		t.Fatalf("EmptyLabel = %q, want %q", resp.EmptyLabel, want)
	}
}
```
