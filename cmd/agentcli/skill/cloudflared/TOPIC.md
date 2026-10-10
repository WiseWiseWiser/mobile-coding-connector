---
name: remote-agent/cloudflared
description: >-
  Switch the origin among native cloudflared, qemu cloudflared, and the edge
  cloudflare-proxy. Republishes, then stops the previous connector.
---

# cloudflared backend

Exactly one backend owns the origin's public hostnames.

| Backend | Runs | Files |
|---------|------|--------|
| `native` | host `cloudflared` | `cloudflare.json` mode cleared, `qemu.json` `enabled: false` |
| `qemu` | guest `cloudflared` | mode cleared, `qemu.json` `enabled: true` |
| `proxy` | no origin `cloudflared`; edge `local-agent cloudflare-proxy` | `mode=proxy`, `proxy_url`, `token`; qemu disabled |

```bash
remote-agent cloudflared status
remote-agent cloudflared use native|qemu|proxy
```

`use` writes both files, republishes every cloudflare hostname (edge
`BindHostname` runs `tunnel route dns` for proxy), checks `/ping`, and only
then stops the previous connector. Stopping first drops `remote-agent`.

`use proxy` reuses the saved `proxy_url` and `token`. It does not rotate the
edge token. Missing either one errors before any file write.

A second `use` of the backend that is already serving is a no-op.

## Wrong → correct

| Symptom | Correct |
|---------|---------|
| `qemu config --disabled` but the public URL stays on the guest | `cloudflared use proxy` (file flag does not move DNS) |
| `mode=proxy` and `qemu.json` enabled at once | `use` writes one pair; do not set the files separately |
| `Error: cloudflare mode=proxy requires proxy_url and token` | Pin both in `cloudflare.json` before `use proxy` |

## Related

| Path | Role |
|------|------|
| `cloudflare-proxy` | Edge `local-agent cloudflare-proxy` (no origin cloudflared) |
| `qemu` | Guest lifecycle; not the backend switch |
