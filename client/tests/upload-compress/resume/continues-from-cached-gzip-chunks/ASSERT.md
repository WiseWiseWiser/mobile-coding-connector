## Expected

1. Upload succeeds with `compressed=true`.
2. `TotalChunkPosts` is 0 (all wire chunks already cached).
3. `ResultSize` equals original content length.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("upload failed: %s", resp.UploadErr)
	}
	if resp.InitCompressed == nil || !*resp.InitCompressed {
		t.Fatal("expected compressed=true")
	}
	if resp.TotalChunkPosts != 0 {
		t.Fatalf("TotalChunkPosts=%d want 0 (resume skipped all cached gzip chunks)", resp.TotalChunkPosts)
	}
	if resp.ResultSize != int64(len(req.Content)) {
		t.Fatalf("ResultSize=%d want %d", resp.ResultSize, len(req.Content))
	}
}
```
