## Expected

- Parse fails with mode required message.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == nil {
		t.Fatal("expected parse error")
	}
	if !errContains(resp.RunErr, "--whitelist or --blacklist") {
		t.Fatalf("err = %v", resp.RunErr)
	}
}
```