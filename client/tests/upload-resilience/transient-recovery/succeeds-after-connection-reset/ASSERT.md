## Expected

1. `UploadErr` is empty — upload succeeds.
2. `TransportAttempts[2]` is 3 (two transport failures + one success).
3. `ChunkAttempts[2]` is 1 (only successful RoundTrip reached server).
4. `TotalChunkPosts` is 5.
5. `InitCount` is 1; `CompleteCalled` is true.
6. `ResultSize` equals `TotalBytes`.

## Side Effects

Assembled file bytes match source after transport-level recovery.

## Errors

- `TransportAttempts[2] < 3`.
- `ChunkAttempts[2] != 1`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("UploadFile failed: %s", resp.UploadErr)
	}
	if resp.TransportAttempts[2] != 3 {
		t.Fatalf("transport attempts for chunk 2 = %d, want 3", resp.TransportAttempts[2])
	}
	if resp.ChunkAttempts[2] != 1 {
		t.Fatalf("server chunk 2 posts = %d, want 1", resp.ChunkAttempts[2])
	}
	if resp.TotalChunkPosts != 5 {
		t.Fatalf("TotalChunkPosts = %d, want 5", resp.TotalChunkPosts)
	}
	if resp.InitCount != 1 {
		t.Fatalf("InitCount = %d, want 1", resp.InitCount)
	}
	if !resp.CompleteCalled {
		t.Fatal("complete endpoint was not called")
	}
	if resp.ResultSize != req.TotalBytes {
		t.Fatalf("ResultSize = %d, want %d", resp.ResultSize, req.TotalBytes)
	}
}
```