## Expected Output

```
Project: local-bound (local-bound-001)
  Dir:              <remote-abs>
  Local Dir:        <local-abs>
  Git Branch:       main
```

## Expected

1. Exit code 0.
2. `Local Dir:` line shows `resp.LocalPath` (absolute).
3. `Local Dir` appears after `Dir:` and before `Git Branch:`.

## Side Effects

None beyond subprocess and temp dirs.

## Errors

- Missing or wrong Local Dir path; dash when binding exists.

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
		t.Fatal("LocalPath not set in response")
	}
	wantLocal := fmt.Sprintf("Local Dir:        %s", resp.LocalPath)
	out := resp.Stdout
	if !strings.Contains(out, "Project: local-bound (local-bound-001)") {
		t.Fatalf("missing project header;\n%s", out)
	}
	if !strings.Contains(out, wantLocal) {
		t.Fatalf("missing %q;\n%s", wantLocal, out)
	}
	if !strings.Contains(out, "Git Branch:       main") {
		t.Fatalf("missing git branch;\n%s", out)
	}
	dirIdx := strings.Index(out, "  Dir:")
	localIdx := strings.Index(out, wantLocal)
	branchIdx := strings.Index(out, "Git Branch:")
	if dirIdx < 0 || localIdx < 0 || branchIdx < 0 || !(dirIdx < localIdx && localIdx < branchIdx) {
		t.Fatalf("field order Dir < Local Dir < Git Branch;\n%s", out)
	}
}
```