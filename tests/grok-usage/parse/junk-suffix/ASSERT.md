## Expected

1. `ParseErr` is empty.
2. `WeeklyLimit` is `59%`.
3. `NextReset` is exactly `July 17, 08:55` — not `… Imag`, not any junk TZ, not invented PT.

## Errors

- Parser invents timezone from scrollback junk, or fails to parse no-TZ date.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.WeeklyLimit != "59%" {
		t.Fatalf("WeeklyLimit = %q, want 59%%", resp.WeeklyLimit)
	}
	want := "July 17, 08:55"
	if resp.NextReset != want {
		t.Fatalf("NextReset = %q, want %q", resp.NextReset, want)
	}
	if strings.Contains(strings.ToLower(resp.NextReset), "imag") {
		t.Fatalf("NextReset must not absorb junk suffix: %q", resp.NextReset)
	}
}
```
