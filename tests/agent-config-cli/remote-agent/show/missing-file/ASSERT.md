## Expected

1. Exit 0.
2. Stdout is pretty-printed JSON with empty domains and empty default.
3. No Config UI banner.

## Side Effects

Does not create the config file (read-only dump).

## Errors

Non-zero exit, invalid JSON, or UI started.

## Exit Code

0.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExitZero(t, resp)
	assertNoConfigUI(t, resp)
	assertPrettyEmptyishConfigJSON(t, resp.Stdout)
}
```
