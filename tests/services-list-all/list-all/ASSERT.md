## Expected

1. `HTTPStatus` is `200`.
2. `ListedIDs` contains both `local-web` and `other-api`.

## Errors

- `?all=1` still filters to server project scope only.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.HTTPStatus != 200 {
		t.Fatalf("status = %d body=%s", resp.HTTPStatus, resp.Body)
	}
	hasLocal, hasOther := false, false
	for _, id := range resp.ListedIDs {
		if id == "local-web" {
			hasLocal = true
		}
		if id == "other-api" {
			hasOther = true
		}
	}
	if !hasLocal || !hasOther {
		t.Fatalf("want both local-web and other-api in list-all response, got %v", resp.ListedIDs)
	}
}
```