---
name: remote-agent/go
description: >-
  GOPROXY on the remote agent plus local fast-fail relay (404 when
  upstream is down so the GOPROXY chain falls through).
---

# Go module proxy

Remote in-process GOPROXY (no token, VPN/internal IP only) plus a Mac
loopback relay hosted by the **local-agent macOS keep-alive** (not
`ssh --serve`). `status` prints the copy-paste recipe.

```bash
remote-agent go mod-proxy status
remote-agent go mod-proxy enable --now --root DIR --port 21000

local-agent go mod-proxy-relay status
local-agent go mod-proxy-relay enable --now --upstream http://10.91.186.143:21000
```

| Fact | Value |
|------|--------|
| Remote | `:21000`, no token |
| Local relay | `127.0.0.1:21001` |
| Upstream | **direct IP** (e.g. `http://10.91.186.143:21000`), not a Cloudflare hostname |
| Down | relay answers **404** so GOPROXY falls through (connection refused does not) |
| Recipe | `GOPROXY=http://127.0.0.1:21001,https://proxy.golang.org,direct` + `GOINSECURE=127.0.0.1` |
| Daemon down | `Error: local agent daemon is not running; start the local-agent macOS app` |

Do **not** mount `GOMODCACHE` over SSHFS/NFS/SMB — see
`$AI/knowledges/ai-critic/remote-go-cache-fs/TOPIC.md`.
