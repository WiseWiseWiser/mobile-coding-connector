## Expected

1. Exit code 0.
2. Combined output mentions binding confirmation (project name or local path).
3. `remote-agent-config.json` contains exactly one `project_bindings` row with
   `remote_dir` equal to the registered project dir and `local_path` equal to the local clone.

## Side Effects

`project_bindings` upserted under isolated agent `HOME`.

## Errors

- Non-zero exit or missing binding row.

## Exit Code

0.

```go
import (
	"path/filepath"
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

	bindings := readConfigBindings(t, resp.RemoteConfigPath)
	if len(bindings) != 1 {
		t.Fatalf("want 1 binding, got %d: %+v", len(bindings), bindings)
	}
	wantRemote, _ := filepath.Abs(req.Project.Dir)
	wantLocal, _ := filepath.Abs(req.LocalPath)
	if bindings[0].RemoteDir != wantRemote {
		t.Fatalf("remote_dir=%q want %q", bindings[0].RemoteDir, wantRemote)
	}
	if bindings[0].LocalPath != wantLocal {
		t.Fatalf("local_path=%q want %q", bindings[0].LocalPath, wantLocal)
	}

	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "bind") && !strings.Contains(combined, filepath.Base(wantLocal)) {
		t.Fatalf("expected confirmation in output:\n%s", resp.Combined)
	}
}
```