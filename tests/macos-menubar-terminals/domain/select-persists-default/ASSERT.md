## Expected

1. `SavedOK` is true.
2. `DefaultServer` equals `https://b.example` (selected domain becomes `default`).
3. `ResolvedOK` is true; `State` is `ok`.
4. `ResolvedServer` is `https://b.example`.
5. `ResolvedToken` is `tok-b` (same resolved endpoint used for Services + Terminals).

## Errors

- Default left on A; resolve still returns A; token dropped on save.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.SavedOK {
		t.Fatal("expected SavedOK after SelectDefaultDomain + Save")
	}
	if resp.DefaultServer != "https://b.example" {
		t.Fatalf("default = %q, want https://b.example", resp.DefaultServer)
	}
	if resp.State != "ok" {
		t.Fatalf("state = %q, want ok", resp.State)
	}
	if !resp.ResolvedOK {
		t.Fatal("expected ResolvedOK=true")
	}
	if resp.ResolvedServer != "https://b.example" {
		t.Fatalf("resolved server = %q, want https://b.example", resp.ResolvedServer)
	}
	if resp.ResolvedToken != "tok-b" {
		t.Fatalf("resolved token = %q, want tok-b", resp.ResolvedToken)
	}
}
```
