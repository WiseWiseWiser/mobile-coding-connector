## Expected

1. `UsesTextStorageAppend` is true (`textStorage` batch append present).

## Side Effects

- None (read-only source inspection).

## Errors

- Hot path only uses `textView.string +=` / full string rewrite per line.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsesTextStorageAppend {
		t.Fatalf("expected textStorage.append (batch write) in BackupProgressWindow; not sole per-line string += (source: %s)", resp.ProgressWindowSource)
	}
}
```
