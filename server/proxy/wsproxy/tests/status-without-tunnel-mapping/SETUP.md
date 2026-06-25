## Preconditions

Local xray responds on `/ws` with HTTP 400, matching production health checks.
`Manager.publicURL` is set to the reported hostname, but no `ws-proxy` ingress
mapping exists on the extension tunnel group (Cloudflare returns 404).

## Steps

1. Enable `SimulateXray`.
2. Set `PublicURL` to `https://ws-25b2a55939e4.xhd2015.xyz`.
3. Leave `AddTunnelMapping` false.

## Context

Reproduces the user report: status shows running and a public URL, yet V2Ray
fails because the Cloudflare tunnel is not routing to xray.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SimulateXray = true
	req.PublicURL = "https://ws-25b2a55939e4.xhd2015.xyz"
	req.InstanceID = "25b2a55939e4"
	req.AddTunnelMapping = false
	return nil
}
```