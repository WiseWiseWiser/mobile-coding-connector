# Scenario

**Feature**: remote-agent config profile

```
# Profile = remote-agent; config file remote-agent-config.json
remote-agent config [...] -> help | JSON | error
HOME/.ai-critic/remote-agent-config.json <- loadConfig / seed
```

## Preconditions

Binary is `./cmd/remote-agent`; CLI name in help is `remote-agent`.

## Steps

1. Set `Profile` to remote-agent.
2. Descendant leaves set `Args` and optional seed.

## Context

Shared `runConfig` with remote profile defaults.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Profile = ProfileRemote
	return nil
}
```
