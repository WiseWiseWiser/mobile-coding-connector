## Expected

1. `HasScrollPolicyHelper` is true (`func ShouldScrollBackupProgressOnFlush`).
2. `ShouldScrollOnFlush` is true (function body `return true`).

## Side Effects

- None (source-contract on helper).

## Errors

- Helper missing or does not return true.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasScrollPolicyHelper {
		t.Fatalf("missing func ShouldScrollBackupProgressOnFlush in macosapp/menubar (checked: %v)", resp.MenubarSourcesChecked)
	}
	if !resp.ShouldScrollOnFlush {
		t.Fatal("ShouldScrollBackupProgressOnFlush must return true (v1 always scroll on flush)")
	}
}
```
