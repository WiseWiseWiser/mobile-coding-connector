## Expected

1. `Token` is `cfg-default-token`.
2. `Source` is `config`.

## Errors

- Falling through to credentials or returning empty when config has a usable token.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "cfg-default-token" {
		t.Fatalf("token = %q, want cfg-default-token", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
