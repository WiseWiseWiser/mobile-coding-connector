# Scenario

**Feature**: an item id derives from its label when --id is omitted

```
add --label "CC v2" -> id "cc-v2"
```

## Preconditions

1. No id is supplied.

## Steps

1. Set `Op=add` and register one item that only has a label.

## Context

Stable ids keep `usage default <id>` and the registry readable.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	req.AddItems = []usageitems.Item{
		usageItem("", "CC v2", usageitems.KindCommandCode, "/tmp/commandcode-v2/.commandcode"),
	}
	return nil
}
```
