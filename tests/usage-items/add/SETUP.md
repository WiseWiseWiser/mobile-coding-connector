# Scenario

**Feature**: registering a usage item through the service API

```
--kind + --label + --home -> default label -> provider fetch -> warnings or strict failure
```

## Preconditions

1. `AddRequest` carries the item plus `Default`, `SkipValidate`, and `Strict`.
2. An item with no id derives it from the label; an item with no label takes the provider name.

## Steps

1. Set `Op=add` and list the items to register in `AddItems`.
2. Point `FailHome` at the directory that must fail to fetch.

## Context

Validation warns by default so a typo in `--home` does not lose the registration;
`--strict` turns the same failure into a rejection.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "add"
	return nil
}
```
