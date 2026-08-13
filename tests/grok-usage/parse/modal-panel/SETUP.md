# Scenario

**Feature**: parse Grok 1.0.3 usage modal panel (crime-scene shape)

```
Weekly limit (plan) + progress bar % + Resets: date
  -> ParseShowUsageOutput -> UsageInfo
```

## Preconditions

`show-usage-modal-panel.txt` fixture — scrollback shape from real Grok 1.0.3
`/usage show` panel (crime scene 2026-08-13): plan subtitle, bar line with
percent, `Resets:` (not legacy `Weekly limit: N%` / `Next reset:`).

## Steps

1. `FixtureFile=show-usage-modal-panel.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/modal-panel`.
Locks desired product behavior after REPRODUCED crime scene
(`Grok: Error: grok process exited: signal: killed` — parser never accepted
panel). Desired: weekly % + bare local reset (no invented TZ).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-usage-modal-panel.txt"
	req.ExpectParseError = false
	return nil
}
```
