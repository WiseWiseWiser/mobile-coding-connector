## Expected

1. `DropdownLine` is exactly `Grok: 6%(Weekly), Reset soon`.
2. Line does not contain `left`.

## Errors

- Appending `, left …` or dropping `Reset soon`.

```go
import "testing"
import "strings"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: 6%(Weekly), Reset soon"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
	if strings.Contains(resp.DropdownLine, "left") {
		t.Fatalf("dropdown must not contain left suffix: %q", resp.DropdownLine)
	}
}
```