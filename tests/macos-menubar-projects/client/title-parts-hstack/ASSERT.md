## Expected

1. `UsesTitlePartsHStack` is true — sources expose title parts (Leading/Trailing
   or `formatProjectTitleParts`) **and** use `HStack` + `Spacer` for left/right
   alignment in the Projects menu.

## Side Effects

- None (read-only source inspection).

## Errors

- Single concatenated `Menu(formatProjectTitle(...))` title only, with no parts API
  and no left/right layout.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.UsesTitlePartsHStack {
		t.Fatalf("expected title parts + HStack/Spacer in Swift sources: %v", resp.SwiftSourcesChecked)
	}
}
```
