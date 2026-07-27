# Scenario

**Feature**: visit provider auto-selection and explicit overrides

```
Start(port, provider, idle) + Available() -> cloudflare_owned | cloudflare_quick | error
```

## Preconditions

Manager leaves only; both fakes registered with leaf-controlled Available.

## Steps

1. Leaf sets OwnedAvailable / QuickAvailable and Provider.
2. Op=visit-start.
3. Assert session.Provider or StartErr.

## Context

auto → owned if Available else quick; explicit owned/quick must be Available.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "auto"
	return nil
}
```
