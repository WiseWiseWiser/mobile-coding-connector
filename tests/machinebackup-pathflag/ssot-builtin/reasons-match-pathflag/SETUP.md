# Scenario

**Feature**: BuiltinExclusionConfig reasons match pathflag catalog reasons

```
for each shared path rule:
  BuiltinExclusionConfig entry.Reason == pathflag.Classify reason golden
```

## Preconditions

- Golden reasons copied from pathflag attributeRules / segment / log suffix / binary product text.

## Steps

1. Op ssot_reasons.
2. Expect ReasonMismatches empty.

## Context

- Locks SSoT text so show-config stays aligned after generation from pathflag.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = OpSSOTReasons
	return nil
}
```
