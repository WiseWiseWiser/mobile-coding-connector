## Expected

1. `HasDoctorHdr` is true (`WS Proxy Doctor` on stdout).
2. `HasCheckLines` is true — multiple lines containing `[ok]`, `[fail]`, `[skip]`, or `[warn]`.
3. `HasResultLine` is true (`Result: healthy` or `Result: unhealthy`).
4. `len(StdoutLines) > 0` before exit — output was not empty until the end.
5. At least one server check line (e.g. `configuration load`) appears in `StdoutLines`.

## Side Effects

Server and remote-agent subprocesses started and torn down.

## Exit Code

Reflects healthy/unhealthy; not asserted in this leaf (see `exit-code-reflects-health`).

## Errors

- Blank stdout until process exit (buffering regression).
- Missing doctor header or check lines.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !resp.HasDoctorHdr {
		t.Fatalf("missing WS Proxy Doctor header; stdout:\n%s", resp.Stdout)
	}
	if !resp.HasCheckLines {
		t.Fatalf("missing [ok]/[fail]/[skip] check lines; stdout:\n%s", resp.Stdout)
	}
	if !resp.HasResultLine {
		t.Fatalf("missing Result line; stdout:\n%s", resp.Stdout)
	}
	if len(resp.StdoutLines) == 0 {
		t.Fatal("stdout was empty — streaming did not produce incremental lines")
	}
	foundConfig := false
	for _, l := range resp.StdoutLines {
		if strings.Contains(strings.ToLower(l), "configuration load") {
			foundConfig = true
			break
		}
	}
	if !foundConfig {
		t.Fatalf("missing configuration load check line; stdout:\n%s", resp.Stdout)
	}
}
```
