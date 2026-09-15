# Scenario

**Feature**: a Command Code item needs --home

```
add --kind commandcode (no --home) -> invalid: --home is required for kind commandcode
```

## Preconditions

1. Command Code items read a sandbox directory, so the home is mandatory.

## Steps

1. Set `Op=add` and register a Command Code item with no home.

## Context

Without a home the provider has no credentials to read.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	req.AddItems = []usageitems.Item{
		usageItem("cc", "CC", usageitems.KindCommandCode, ""),
	}
	return nil
}
```
