## Expected

1. `HTTPStatus` is `200`.
2. `ListedIDs` contains `local-web`.
3. `ListedIDs` does not contain `other-api`.

## Errors

- Cross-project service leaked into default list.

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
	if !hasLocal {
		t.Fatalf("missing local-web in %v", resp.ListedIDs)
	}
	if hasOther {
		t.Fatalf("other-api should not appear in scoped list: %v", resp.ListedIDs)
	}
}
```