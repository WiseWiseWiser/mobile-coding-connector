## Expected

1. `UploadErr` is empty — upload completes after session loss.
2. `InitCount` is at least 2 (new session created after 404).
3. `CompleteCalled` is true.
4. `ResultSize` equals `TotalBytes`.

## Side Effects

All 40 chunks assembled on server despite mid-upload session invalidation.

## Errors

- `UploadErr` contains `upload session not found`.
- `InitCount < 2`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("upload should recover after session 404, got: %s", resp.UploadErr)
	}
	if resp.InitCount < 2 {
		t.Fatalf("InitCount=%d, want >=2 (re-init after session loss)", resp.InitCount)
	}
	if !resp.CompleteCalled {
		t.Fatal("complete should be called after recovery")
	}
	if resp.ResultSize != req.TotalBytes {
		t.Fatalf("ResultSize=%d want %d", resp.ResultSize, req.TotalBytes)
	}
}
```