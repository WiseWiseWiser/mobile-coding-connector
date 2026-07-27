# Scenario

**Feature**: invalid dir payloads return 4xx with error envelope

```
POST open {bad dir} -> 4xx {"error":"..."} ; Open skipped (validate first)
# 5xx reserved for Open/osascript failures after a valid directory
```

## Preconditions

Injected Open records calls; validation rejects before Open.

## Steps

1. Leaf sets invalid dir condition.
2. Assert **4xx** and non-empty error.

## Context

REQUIREMENT scenarios 3–4 + locked decision: client/validation → 4xx only.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.OmitMode = true
	req.OmitSend = true
	return nil
}
```
