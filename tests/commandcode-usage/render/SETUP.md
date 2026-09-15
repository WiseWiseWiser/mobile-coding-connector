# Scenario

**Feature**: menu-bar title suffix and dropdown body

```
Response -> TitleSuffix -> "5%"
Response -> FormatBody  -> "5% used, $66.19 left, ..."
```

## Preconditions

1. The label prefix is added by the usage item renderer, not here.

## Steps

1. Set `Op=render` for a ready account.

## Context

The menu-bar title metric for Command Code is the cycle credit percent.

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
