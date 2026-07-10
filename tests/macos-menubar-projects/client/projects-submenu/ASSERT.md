## Expected

1. `HasProjectsMenu` is true — local `AICriticApp.swift` includes a Projects menu.

## Side Effects

- None (read-only source inspection).

## Errors

- Projects menu not wired in the local menubar app.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasProjectsMenu {
		t.Fatalf("Projects submenu not found in Swift sources: %v", resp.SwiftSourcesChecked)
	}
}
```
