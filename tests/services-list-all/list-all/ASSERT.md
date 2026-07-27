---
explanation: "L2 Manager.ListAll cross-project"
---

## Expected

1. `HTTPStatus` is `200`.
2. `ListedIDs` contains both `local-web` and `other-api`.

## Errors

- ListAll still filters to server project scope only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
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
