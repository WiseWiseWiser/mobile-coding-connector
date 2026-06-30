## Expected Output

CLI stdout contains the running-service disable prompt.

## Expected

1. `Response.ExitCode` is 0.
2. `Response.Stdout` contains `won't stop immediately` (case-insensitive).
3. On-disk `services.json` has `enabled: false` for the target.

## Side Effects

- CLI prints API `message` to stdout.

## Errors

- Non-zero CLI exit code.
- Missing prompt in stdout.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		combined := ""
		if resp != nil {
			combined = resp.Combined
		}
		t.Fatalf("Run error: %v\ncombined:\n%s", err, combined)
	}
	if resp.ActionError != "" {
		t.Fatalf("CLI run failed: %s", resp.ActionError)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assert.Output(t, strings.ToLower(resp.Stdout), `<contains>
won't stop immediately
</contains>`)

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || *enabled {
		t.Fatalf("services.json enabled = %v present=%v, want false", enabled, present)
	}
}
```