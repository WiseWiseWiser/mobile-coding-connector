# Scenario

**Feature**: no-TZ Next reset must not treat following junk as timezone

```
Next reset: July 17, 08:55\n Imag -> UsageInfo reset ends with PT only
```

## Preconditions

`show-usage-junk-suffix.txt` — date/time then newline then junk word `Imag`
(catches false match of catch-all `[A-Z]{2,4}` across whitespace).

## Steps

1. `FixtureFile=show-usage-junk-suffix.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/junk-suffix`.
Classic TDD: RED against current `nextResetRe` which allows `[A-Z]{2,4}` and can
capture `Imag` as a timezone. Desired: no-TZ candidate wins; bare local wall clock.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-junk-suffix.txt"
	req.ExpectParseError = false
	return nil
}
```
