## Expected Output

Daemon or server logs should show `Server is ready` or successful `/ping` handling
before extension task lines.

## Expected

1. `Response.PingBeforeExt` is true.
2. `Response.ServerReady` is true.
3. Merged logs contain extension work signal (`extension_start` or
   `[auto-task] Running extension` or legacy startup tasks) **after** readiness.

## Side Effects

- Harness polls `/ping` concurrently with daemon startup.

## Errors

- `/ping` never succeeds — fail.

## Exit Code

- `0` when ping precedes extension logs.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ServerReady {
		t.Fatal("daemon never saw ready server")
	}
	if !resp.PingBeforeExt {
		t.Fatal("/ping did not succeed before extension task logs")
	}
}
```