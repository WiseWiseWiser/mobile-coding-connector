# Scenario

**Feature**: a ready account renders the menu-bar title and dropdown body

```
TitleSuffix -> "5%"
FormatBody  -> "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
```

## Preconditions

1. The fixture API reports the ready account.

## Steps

1. Set `Op=render`.

## Context

The usage item renderer prefixes the label; this type supplies the percent and body.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "render"
	return nil
}
```
