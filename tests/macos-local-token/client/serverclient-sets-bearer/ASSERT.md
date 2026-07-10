## Expected

1. At least one Swift source file is found (`SourcesChecked` non-empty).
2. `SetsAuthorization` is true — request path sets an Authorization header.
3. `UsesBearerScheme` is true — Bearer scheme is applied with token plumbing.
4. `OmitsBareBearerEmpty` is true — empty token does not force bare `Bearer `.

## Side Effects

- None (read-only source inspection).

## Errors

- ServerClient still uses unauthenticated `session.data(from:)` with no Authorization.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.SourcesChecked) == 0 {
		t.Fatal("no ServerClient/LocalAuth Swift sources found under macos-ai-critic")
	}
	if !resp.SetsAuthorization {
		t.Fatalf("sources do not set Authorization header (checked: %v)", resp.SourcesChecked)
	}
	if !resp.UsesBearerScheme {
		t.Fatalf("sources do not apply Bearer scheme with token (checked: %v)", resp.SourcesChecked)
	}
	if !resp.OmitsBareBearerEmpty {
		t.Fatalf("sources appear to force bare Bearer without empty-token guard (checked: %v)", resp.SourcesChecked)
	}
}
```
