# Scenario

**Feature**: local-agent config profile

```
# Profile = local-agent; config file local-agent-config.json
local-agent config [...] -> help | JSON | error
HOME/.ai-critic/local-agent-config.json <- loadConfig / seed
```

## Preconditions

Binary is `./cmd/local-agent`; CLI name in help is `local-agent`.

## Steps

1. Set `Profile` to local-agent.
2. Descendant leaves set `Args` and optional seed.

## Context

Shared `runConfig` with local profile; path isolation from remote-agent-config.json.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Profile = ProfileLocal
	return nil
}
```
