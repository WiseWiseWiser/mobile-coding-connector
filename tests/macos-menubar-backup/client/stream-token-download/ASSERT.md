## Expected

1. `UsesStreamTokenDownload` is true (stream endpoint + `archive_token` / archiveToken).

## Side Effects

- None (read-only source inspection).

## Errors

- Only local file copy, or non-stream JSON path without token download.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsesStreamTokenDownload {
		t.Fatalf("missing stream+archive_token download contract (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
