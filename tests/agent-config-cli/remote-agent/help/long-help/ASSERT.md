## Expected

1. Exit 0.
2. Help mentions `remote-agent`, `--web`, and `--show`.
3. No Config UI banner.

## Side Effects

None.

## Errors

Missing flags or non-zero exit.

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
	assertHelpMentionsFlags(t, resp.Stdout, "remote-agent")
}
```
