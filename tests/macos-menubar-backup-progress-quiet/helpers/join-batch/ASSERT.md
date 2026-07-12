## Expected

1. `HasJoinBatchHelper` is true (`func JoinBackupProgressBatch` in `macosapp/menubar`).
2. `JoinBatchPolicyOK` is true — body implements:
   - empty / nil → `return ""`
   - non-empty → `strings.Join(..., "\n")` with trailing `"\n"`
3. Documented pure results (implementer must match):
   - `["a","b"]` → `"a\nb\n"`
   - `[]` / nil → `""`
   - `["solo"]` → `"solo\n"`

## Side Effects

- None (source-contract on helper; pure call can replace this once symbol exists).

## Errors

- Missing function, or join policy without empty guard / newline join / trailing newline.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasJoinBatchHelper {
		t.Fatalf("missing func JoinBackupProgressBatch in macosapp/menubar (checked: %v)", resp.MenubarSourcesChecked)
	}
	if !resp.JoinBatchPolicyOK {
		t.Fatalf("JoinBackupProgressBatch body must empty→\"\" and strings.Join + trailing \\n (checked: %v)", resp.MenubarSourcesChecked)
	}
}
```
