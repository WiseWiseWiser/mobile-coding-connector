## Expected

1. `UsesStreamTokenDownload` is true (stream endpoint + `archive_token` / archiveToken).

## Side Effects

- None (read-only source inspection).

## Errors

- Only local file copy, or non-stream JSON path without token download.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsesStreamTokenDownload {
		t.Fatalf("missing stream+archive_token download contract (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
