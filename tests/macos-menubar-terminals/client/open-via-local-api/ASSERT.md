## Expected

1. `LocalTerminalsUseOpenAPI` is true — local attach/new path uses openITerm2
   and `/api/local/iterm2/open`.
2. `HasDirectITermOpenerProductPath` is false.

## Side Effects

- None (read-only source inspection).

## Errors

- Still `ITermOpener.openCommandOrAlert(cmd)` only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.LocalTerminalsUseOpenAPI {
		t.Fatalf("local terminals must open via /api/local/iterm2/open + openITerm2 (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.HasDirectITermOpenerProductPath {
		t.Fatalf("local terminals still use ITermOpener product path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
