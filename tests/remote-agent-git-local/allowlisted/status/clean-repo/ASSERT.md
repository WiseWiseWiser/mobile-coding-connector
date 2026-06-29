## Expected

1. Exit 0.
2. Stdout contains branch `main` and indicates a clean working tree.

## Side Effects

None beyond streaming.

## Errors

- Missing branch line or dirty indicators on clean repo.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	assert.Output(t, resp.Stdout, `<contains>
<any-of><expect>On branch main</expect><expect>## main</expect></any-of>
nothing to commit, working tree clean
</contains>`)
}
```