# Scenario

**Feature**: full-stack streaming doctor via remote-agent subprocess

```
# build server + agent, seed ws-proxy config, run doctor, capture stdout pipe
ai-critic-server -> /doctor/stream -> remote-agent ws-proxy doctor -> stdout lines
```

## Preconditions

1. `go build` can produce `ai-critic-server` and `remote-agent` binaries.
2. Isolated `AI_CRITIC_HOME` with test credentials.
3. ws-proxy config seeded; network checks stubbed for speed.

## Steps

1. Root `Run` builds binaries, starts server, waits for `/ping`.
2. Runs `remote-agent --server URL --token testpassword ws-proxy doctor`.
3. Reads stdout line-by-line until process exit; records timestamps when requested.

## Context

Primary integration guarantee for the streaming progress feature. Complements
unit trees under `server/proxy/wsproxy/tests/streaming-doctor/`,
`client/tests/streaming/`, and `cmd/agentcli/streamcmd/tests/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.UpstreamFetchDelayMs < 0 {
		req.UpstreamFetchDelayMs = 0
	}
	return nil
}
```
