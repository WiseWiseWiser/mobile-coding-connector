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
Covers multi-format Next reset (REQUIREMENT-DESIGN-grok-usage-next-reset-multi-format).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "parse"
	return nil
}
```