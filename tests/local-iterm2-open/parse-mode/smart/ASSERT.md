## Expected

1. No parse error.
2. `ParsedMode` is `ModeSmart`.

```go
import (
	"testing"

	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("ParseErr = %q", resp.ParseErr)
	}
	if resp.ParsedMode != iterm2.ModeSmart {
		t.Fatalf("ParsedMode = %v, want ModeSmart", resp.ParsedMode)
	}
}
```
