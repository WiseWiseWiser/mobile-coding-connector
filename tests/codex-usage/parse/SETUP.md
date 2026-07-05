# Scenario

**Feature**: ParseStatusOutput from fixture stdout

```
fixture stdout -> ParseStatusOutput -> UsageInfo or error
```

## Preconditions

Fixture files under shared `testdata/`.

## Steps

1. Set `Op=parse` in leaf setup.

## Context

Pure parser tests; no daemon or network.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "parse"
	return nil
}
```