---
explanation: "L2 agentcli service add happy path"
---

## Expected Output

Stdout contains a Created line and the service name; ends with trailing newline.

## Expected

1. `ExitCode` is 0.
2. Stdout contains `Created` (case-insensitive) and `demo-add`.
3. Stdout ends with `\n`.
4. `ServicesOnDisk` has a row with name `demo-add` and non-empty id.
5. `ListedNames` (ListAll) contains `demo-add`.

## Side Effects

- `services.json` gains one definition.
- Process is **not** required to be running (definition-only default).

## Errors

- Non-zero exit (unknown subcommand until implemented).
- Missing Created / name / disk row.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		combined := ""
		if resp != nil {
			combined = resp.Combined
		}
		t.Fatalf("Run error: %v\ncombined:\n%s", err, combined)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := strings.ToLower(resp.Stdout)
	if !strings.Contains(out, "created") {
		t.Fatalf("stdout missing Created; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "demo-add") {
		t.Fatalf("stdout missing name demo-add; stdout:\n%s", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout must end with newline; got %q", resp.Stdout)
	}
	if !diskHasName(resp.ServicesOnDisk, "demo-add") {
		t.Fatalf("services.json missing demo-add; disk=%v", resp.ServicesOnDisk)
	}
	// non-empty id on disk
	foundID := false
	for _, row := range resp.ServicesOnDisk {
		if n, _ := row["name"].(string); n == "demo-add" {
			if id, _ := row["id"].(string); id != "" {
				foundID = true
			}
		}
	}
	if !foundID {
		t.Fatalf("demo-add row missing non-empty id; disk=%v", resp.ServicesOnDisk)
	}
	if !listContainsName(resp.ListedNames, "demo-add") {
		t.Fatalf("ListAll missing demo-add; names=%v", resp.ListedNames)
	}
}
```
