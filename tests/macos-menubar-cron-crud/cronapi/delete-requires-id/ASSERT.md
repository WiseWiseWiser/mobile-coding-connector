## Expected

1. `BuildOK` is false.
2. `BuildErr` is non-empty (id required).

## Errors

- Empty id accepted; silent bad URL.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.BuildOK {
		t.Fatal("expected error for empty task id")
	}
	if resp.BuildErr == "" {
		t.Fatal("BuildErr empty")
	}
}
```
