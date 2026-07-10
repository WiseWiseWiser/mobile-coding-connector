# Scenario

**Feature**: open path is iTerm-only via local open API (no Terminal.app; no raw osascript product path)

```
session click / New Terminal
  -> ServerClient.openITerm2 -> POST /api/local/iterm2/open (mode + send)
missing iTerm -> error/alert from server/client; never open /Applications/Terminal.app
local product must not call ITermOpener.openCommandOrAlert for terminals
```

## Preconditions

Local app sources under `macos-ai-critic/ai-critic-macos/`. Shared may still
contain thin helpers, but **local** product terminal open must go through HTTP
open API. Remote app may keep client-side iTerm open (not required to call
`/api/local/iterm2/open`).

## Steps

1. Set `ClientLeaf=iterm-only`.

## Context

REQUIREMENT leaf: iTerm-only + terminals refactor onto `/api/local/iterm2/open`.
**Updated** from prior contract that only checked iTerm string presence.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "iterm-only"
	return nil
}
```
