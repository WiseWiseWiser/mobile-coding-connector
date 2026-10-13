---
name: remote-agent/port
description: >-
  Publish a server port at a public hostname: persistent `port mapping
  add/list/update/delete`, short-lived `port visit`, and `port list` for what
  is listening. Owned Cloudflare domain, no origin cloudflared needed.
---

# Port

Two ways to make a port public. Pick by lifetime, not by convenience:

| Want | Use | Hostname | Lifetime |
|------|-----|----------|----------|
| A stable address, survives restarts | **`port mapping`** | You choose | Until `delete` |
| A one-off URL to show someone | **`port visit`** | Random | Idle timeout (`--idle`, default 10m) |

`port visit` **cannot** be given a hostname: the server hardcodes an ephemeral
subdomain for ad-hoc visits and never records the name. If the user named the
domain they want, it is `mapping`.

## mapping

```bash
local-agent port mapping add 9999 --hostname house-design-aes821.xhd2015.xyz
local-agent port mapping list
local-agent port mapping update 9999 --hostname house-design-v2.xhd2015.xyz
local-agent port mapping delete 9999              # keeps the saved name
local-agent port mapping delete 9999 --forget     # also clears it
```

```console
$ local-agent port mapping add 9999 --hostname house-design-aes821.xhd2015.xyz
mapping added
  Port:      9999
  Hostname:  house-design-aes821.xhd2015.xyz
  Provider:  cloudflare_owned
  URL:       https://house-design-aes821.xhd2015.xyz
  Name:      saved (--forget on delete to clear)
```

`list` shows live mappings **and** ports that only have a saved name. The
second kind is not dead state: it is what `delete` leaves behind, and what the
next `add` reuses so nobody retypes a domain.

```console
$ local-agent port mapping list
PORT    STATUS    PROVIDER           PUBLIC URL
3080    inactive  -                  (saved name: dsh-on-mac-aes123.xhd2015.xyz)
8194    active    cloudflare_owned   https://mac-agent-run.xhd2015.xyz
9393    inactive  -                  (saved name: xiangqi.xhd2015.xyz)
23712   active    cloudflare_domain  https://mac-agent-aes42.xhd2015.xyz

4 mapping(s)
```

### Semantics

| Rule | Detail |
|------|--------|
| `add` records the name **before** publishing | Mirrors the web UI, so the hostname survives a later `delete` |
| `delete` keeps the name | The forward goes; the name stays until `--forget`. This is why orphan rows exist in `list` |
| `update` = remove + add | The server refuses a second mapping on one port (`port N is already being forwarded`), so `add` on a mapped port tells you to use `update` |
| `update` rolls back on failure | Old mapping is re-added, so a typo cannot leave the port unpublished |
| A hostname on another port is **taken over** | The stale mapping is dropped silently. Check `list` first if that was not the intent |
| `--provider` | `owned` (default) and `quick`, or canonical `cloudflare_owned` / `cloudflare_quick` |

```console
$ local-agent port mapping add 9999 --hostname h.xhd2015.xyz
warning: port 9999 is not listening; the mapping is published but will not serve until something binds it
mapping added
  ...
```

```console
$ local-agent port mapping add 9999 --hostname a.xhd2015.xyz
Error: port 9999 is already being forwarded; use 'port mapping update 9999 --hostname a.xhd2015.xyz'
```

```console
$ local-agent port mapping update 9999 --hostname new.xhd2015.xyz
Error: port 9999 is not mapped; use 'port mapping add 9999 --hostname new.xhd2015.xyz'
```

## visit

```bash
local-agent port visit 9999 --detach      # print a URL and return
local-agent port visit list
local-agent port visit stop <id|port>
```

Without `--detach` the command blocks until the visit expires.

## list

```bash
local-agent port list                 # PORT / PID / COMMAND
local-agent port list --forwards      # also the persistent mappings
```

## Publishing returns before the edge is ready

`add` succeeds once the tunnel ingress is written, but the hostname can answer
`530` / `error code: 1033` for a short while as Cloudflare propagates. Retry
before debugging:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' https://<hostname>/ping
```

`POST /api/ports/restart-dns` fixes DNS for **tunnel groups** (the `core`
group). A `cloudflare_owned` mapping published by `port mapping` lives in the
**managed tunnel** under `~/.cloudflared/managed-tunnels/ai-critic-*`, which is
a different tunnel — so restart-dns is not the lever here. Confirm the ingress
landed in that managed config, and let propagation finish.

## Related

| Path | Role |
|------|------|
| topic **`deploy-service`** | Standing up a *new* service (install → upload → `service add --port-provider …`) |
| topic **`cloudflared`** | Switching the ai-critic origin backend (native / qemu / proxy) |
| topic **`cloudflare-proxy`** | Origin behind an edge proxy, no origin cloudflared |

**Note.** Topics are `go:embed`-ed into the binary: an already-installed
`local-agent` keeps showing the old topic list until it is rebuilt.
