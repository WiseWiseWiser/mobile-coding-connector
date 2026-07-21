## Expected

1. Exit 0 within timeout.
2. Help uses `local-agent` branding and documents `--web` / `--show`.
3. No `Config UI running`.

## Side Effects

None.

## Errors

Timeout (UI path), wrong binary name, missing flags.

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
	assertHelpMentionsFlags(t, resp.Stdout, "local-agent")
}
```
