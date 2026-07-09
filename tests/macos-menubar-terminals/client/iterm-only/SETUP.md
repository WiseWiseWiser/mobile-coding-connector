# Scenario

**Feature**: open path is iTerm-only with no Terminal.app fallback

```
session click / New Terminal -> iTerm2 (ModeForceNew + follow-up cmd)
missing iTerm -> error/alert; never open /Applications/Terminal.app
```

## Preconditions

Both apps (and Shared open helpers) must not fall back to Apple Terminal.

## Steps

1. Set `ClientLeaf=iterm-only`.

## Context

REQUIREMENT leaf: `client/remote-no-terminal-app-fallback` / iTerm-hardcoded.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "iterm-only"
	return nil
}
```
