# Scenario

**Feature**: preferred model substring per agent id

```
applyPreferredModel -> substring for grok is grok not kimi-k2.5
```

## Preconditions

- Export helper `TestExported_PreferredModelSubstringForAgent`.

## Steps

1. `Op = OpModelSubstring`.

## Context

Requirement: grok sessions prefer model IDs containing grok.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpModelSubstring
	return nil
}
```