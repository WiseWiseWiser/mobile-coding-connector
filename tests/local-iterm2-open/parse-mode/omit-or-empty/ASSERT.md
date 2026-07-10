## Expected

1. `ParseErr` is empty.
2. `ParsedMode` equals `iterm2.ModeReuseCurrent`.

## Errors

- Mapping empty to `ModeSmart` (lib zero-value trap).

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
		t.Fatalf("ParseErr = %q, want empty", resp.ParseErr)
	}
	if resp.ParsedMode != iterm2.ModeReuseCurrent {
		t.Fatalf("ParsedMode = %v, want ModeReuseCurrent (%v)", resp.ParsedMode, iterm2.ModeReuseCurrent)
	}
}
```
