# Scenario

**Feature**: tty.ParseShowUsageOutput from fixture scrollback

```
fixture scrollback -> tty.ParseShowUsageOutput -> multi-format Next reset (first match) -> UsageInfo or error
```

## Preconditions

Fixture files under shared `testdata/`. Parser source of truth is `agent/grok/tty`
(`parseUsageText` / ordered Next-reset candidates: PT, UTC, no-TZ→bare local wall clock).

## Steps

1. Set `Op=parse` in leaf setup.

## Context

Pure parser tests; no daemon, network, or PTY fetch.
Covers multi-format Next reset (REQUIREMENT-DESIGN-grok-usage-next-reset-multi-format)
and Grok 1.0.3 modal panel (`Resets:` / plan subtitle) from crime-scene REPRODUCED.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "parse"
	return nil
}
```