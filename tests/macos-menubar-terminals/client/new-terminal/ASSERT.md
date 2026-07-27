## Expected

1. `HasNewTerminal` is `true` — both local and remote sources mention New Terminal.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing on one product, or only via a cwd prompt flow (not required for this leaf).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNewTerminal {
		t.Fatalf("New Terminal missing on local and/or remote (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
