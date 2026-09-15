## Expected

1. Status is 200.
2. The payload carries the selection state and per-item rendered text.
3. The JSON keys match what the macOS app decodes.

## Errors

- Returning bare usage data, or renaming the rendered fields.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.APIStatus != 200 {
		t.Fatalf("status = %d, body = %s", resp.APIStatus, resp.APIBody)
	}
	if resp.Default != "grok" || !resp.Rotate || len(resp.Items) != 2 {
		t.Fatalf("payload = %s", resp.APIBody)
	}
	for _, want := range []string{`"version":1`, `"default":"grok"`, `"rotate":true`, `"title":"Grok 61%"`, `"dropdown":"Grok: 61%(Weekly), Reset July 17, 08:55, left 4d"`} {
		if !strings.Contains(resp.APIBody, want) {
			t.Fatalf("payload missing %s:\n%s", want, resp.APIBody)
		}
	}
}
```
