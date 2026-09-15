# Scenario

**Feature**: rendered menu-bar title and dropdown text per item

```
Snapshot -> FormatTitle / FormatDropdown -> ItemView{title, dropdown, detail, usage_url}
```

## Preconditions

1. Every rendered string comes from the server, not the app.
2. Disabled items are never fetched and stay `loading`.

## Steps

1. Set `Op=list`, seed `SeedItems`, and choose `SeedDefault` / `SeedRotate`.

## Context

The macOS app prints `title` and `dropdown` verbatim, so these leaves are the
user-visible contract for the two Command Code items.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list"
	return nil
}
```
