## Expected

1. Exit 0 (not an error for combining --json with --show).
2. Stdout matches the same pretty config content as --show alone.

## Side Effects

None beyond read.

## Errors

Rejects --json, wrong content, or non-zero exit.

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
	assertPrettyConfigMatches(t, resp.Stdout, sampleRemoteConfig())
}
```
