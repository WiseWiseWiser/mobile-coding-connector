## Expected

1. Exit 0.
2. Stdout shows staged diff with `-v1` / `+v2` (or equivalent hunk).

## Side Effects

None.

## Errors

- Empty stdout when staged change exists.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s", resp.ExitCode, resp.Stdout)
	}
	assert.Output(t, resp.Stdout, `<contains>
diff --git a/file.txt b/file.txt
@@
-v1
+v2
</contains>`)
}
```