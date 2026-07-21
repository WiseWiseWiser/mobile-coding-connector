## Expected

1. Exit code 0 (process must finish within harness timeout).
2. Stdout is help text for `remote-agent config` mentioning `--web` and `--show`.
3. Combined output does not contain `Config UI running`.

## Side Effects

No long-lived HTTP listener for the config UI.

## Errors

- Process times out (still opens UI).
- Exit non-zero.
- Help missing new flags or UI banner printed.

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
