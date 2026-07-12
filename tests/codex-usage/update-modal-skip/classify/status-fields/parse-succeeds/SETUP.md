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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "05-status-fields.snapshot.txt"
	return nil
}
```
