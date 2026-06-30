## Expected

1. Exit code 0.
2. Single project block with bound `Local Dir` path.

## Side Effects

None.

## Errors

- Missing Local Dir on git-config get output.

## Exit Code

0.

```go
import (
	"fmt"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.LocalPath == "" {
		t.Fatal("LocalPath not set")
	}
	wantLocal := fmt.Sprintf("Local Dir:        %s", resp.LocalPath)
	out := resp.Stdout
	if !strings.Contains(out, "Project: local-git-config (local-gcfg-001)") {
		t.Fatalf("missing project header;\n%s", out)
	}
	if !strings.Contains(out, wantLocal) {
		t.Fatalf("missing %q;\n%s", wantLocal, out)
	}
}
```