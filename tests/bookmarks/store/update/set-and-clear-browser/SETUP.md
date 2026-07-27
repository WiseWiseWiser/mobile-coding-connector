# Scenario

**Feature**: set browser then clear so inheritance applies again

```
Update browser=firefox then ClearBrowser -> browser nil/empty
```

## Preconditions

1. Seed bm_b; update set browser then SecondOp clear-browser.

## Steps

1. Set browser firefox.
2. Clear browser.
3. Assert browser is nil or empty string pointer.

## Context

Optional per-bookmark browser field.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PreAdds = []SeedNode{{
		Type: "url", ID: "bm_b", Name: "B", URL: "https://b.example.com",
	}}
	req.ID = "bm_b"
	req.SetBrowser = true
	req.Browser = "firefox"
	req.SecondOp = "clear-browser"
	return nil
}
```
