## Expected

1. `UploadErr` is empty.
2. `TotalChunkPosts` is 1 (only missing chunk 39 uploaded).
3. `CompleteCalled` is true.

## Side Effects

Re-upload does not re-transfer the 39 cached chunks (~73% of file).

## Errors

- `TotalChunkPosts > 1` (full re-upload bug).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("upload failed: %s", resp.UploadErr)
	}
	if resp.TotalChunkPosts != 1 {
		t.Fatalf("TotalChunkPosts=%d, want 1 (skip 39 cached chunks); full re-upload bug", resp.TotalChunkPosts)
	}
	if !resp.CompleteCalled {
		t.Fatal("complete should succeed after uploading missing chunk only")
	}
}
```