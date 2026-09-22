---
name: remote-agent/cloudflare-proxy
description: >-
  Publish an origin ai-critic over a remote cloudflare-proxy (no cloudflared
  on the origin). HTTP and WebSocket (bash, event-bus) use TCP-over-WS.
---

# Cloudflare proxy

Origin **does not** run `cloudflared`. An **edge** host runs
`local-agent cloudflare-proxy`; origin `cloudflare.json` uses `mode=proxy`.
Visitor HTTP and WebSocket (including `remote-agent bash`) are TCP-piped
over the edge `/dial` sockets.

```text
Visitor → Cloudflare → edge :23790 → /dial WS pool → origin :23712
```

| Role | Where |
|------|--------|
| Edge | `local-agent cloudflare-proxy start --foreground --domain proxy.example.com` (`127.0.0.1:23790`) |
| Origin | `cloudflare.json`: `mode=proxy`, `proxy_url`, `token`; `qemu.json` `enabled: false` |
| App host | `https://app.example.com` (Host mapping on the edge tunnel) |
| Named tunnel | `ai-critic-<hostname>` on the **edge** (never reuse another machine’s tunnel) |

`remote-agent proxy` is **HTTP proxy list**, not this.

## CLI

```text
$ local-agent cloudflare-proxy start --foreground --domain proxy.example.com
listen:     127.0.0.1:23790  running
domain:     proxy.example.com
auth:       token
```

Pin `token` in edge `~/.ai-critic/cloudflare-proxy/config.json` so restarts
do not rotate. Copy that value into origin `cloudflare.json` (do not commit it).

Default pool is **8**. One visitor request (or one WS session) consumes a slot
until the origin TCP closes; workers refill.

## Wrong → correct

| Symptom | Correct |
|---------|---------|
| `bash`: 400 Failed to upgrade | Both binaries TCP-over-WS; edge log `GET /api/terminal ws 101` |
| 503 `no connected origin` | Origin `Publish` workers not connected; restart origin after edge restart |
| 409 hostname already mapped | `cloudflare-proxy delete` then republish |
| `/ping` 200, WS 400 | Old JSON-HTTP proxy; upgrade **edge and origin** together |
| Empty `tunnel_name` steals a shared CF tunnel | Persist `ai-critic-<hostname>` on the edge |

## Verify

```bash
curl -sS https://app.example.com/ping    # 200 pong
remote-agent bash
```

## Related

| Path | Role |
|------|------|
| `proxy` | `remote-agent proxy list` (different command) |
| `$AI/knowledges/ai-critic/cloudflare-proxy/TOPIC.md` | Fleet hosts (private) |
