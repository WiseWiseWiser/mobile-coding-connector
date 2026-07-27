# Scenario

**Feature**: exited status match is case-insensitive and trims surrounding space

```
FormatTerminalTitle("demo","abc"," Exited ") -> "demo [EXITED]"
```

## Preconditions

Status is a mixed-case / padded form of `exited` (not the literal lowercase token).

## Steps

1. Set name `demo`, session id `abc`, status ` Exited ` (leading/trailing space, mixed case).

## Context

REQUIREMENT: match status with trim + equal fold to `exited`; suffix still exact ` [EXITED]`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "demo"
	req.SessionID = "abc"
	req.Status = " Exited "
	return nil
}
```
