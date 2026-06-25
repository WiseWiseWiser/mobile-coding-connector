## Preconditions

Server restarted: xray still listens locally, but in-memory `publicURL` was lost.

## Steps

1. Enable `SimulateXray`.
2. Set `ClearPublicURL` true.
3. Persist `InstanceID`, `Subdomain`, and `UUID` in ws-proxy.json.

## Context

After restart, `Status()` can report `Running: true` via `isXrayAlive` while
`publicURL` stays empty, so `vmess-link` fails even though xray is listening.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SimulateXray = true
	req.ClearPublicURL = true
	req.InstanceID = "25b2a55939e4"
	req.Subdomain = "ws"
	return nil
}
```