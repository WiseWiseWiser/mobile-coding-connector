# Scenario

**Feature**: ParseStatusSnapshot succeeds on live status chrome

```
05-status-fields.snapshot.txt -> MonthlyUsage / CreditsUsed / CreditsTotal / NextReset
```

## Preconditions

Fixture contains `Monthly credit limit: … % left`, `N of M credits used`, `(resets …)`.

## Steps

1. `FixtureFile=05-status-fields.snapshot.txt`.

## Context

Account-specific numbers are from the capture day; assert exact fixture values.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "05-status-fields.snapshot.txt"
	return nil
}
```
