# Scenario

**Feature**: bookmark menu title uses node name

```
FormatBookmarkMenuTitle("Local Dashboard") -> "Local Dashboard"
```

## Preconditions

1. LabelKind url_title; LabelName set.

## Steps

1. Format title.
2. Exact name.

## Context

URL item title in submenu.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.LabelKind = "url_title"
	req.LabelName = "Local Dashboard"
	return nil
}
```
