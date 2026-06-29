## Expected

1. Exit 0.
2. Stdout marks `main` as the current branch (`* main` or equivalent).

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
<any-of><expect>* main</expect><expect>main *</expect></any-of>
</contains>`)
}
```