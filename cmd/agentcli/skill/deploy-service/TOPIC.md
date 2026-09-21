---
name: remote-agent/deploy-service
description: >-
  Stand up a long-lived web service on the remote with a public domain:
  probe/install binary, dir-upload data, service add with port-forward,
  verify. Recipe composing topics install, upload, service.
---

# Deploy Service

End-to-end recipe: local Go CLI + data → managed remote service → public domain.

## 1. Binary (conditional)

```bash
remote-agent exec sh -c '<cmd> --help >/dev/null 2>&1' \
  && echo current || remote-agent install <cmd>
```

Probe first: skips a frontend build + ~50 MB upload when the remote copy fits. See topic **install**.

## 2. Data (dir upload)

```bash
remote-agent upload ./datadir /root/datadir        # dest absent → contents land directly
remote-agent exec rm -f /root/datadir/server.json  # drop process-owned files
```

| Rule | Detail |
|------|--------|
| Dest absent | created; **contents** copied in (no nesting) |
| Dest exists | cp -R **nests** `<dest>/<basename>` — refresh: `exec rm -rf <dest>` first |
| Process-owned files | upload can't exclude; `exec rm` after apply (discovery records) |

## 3. Service + domain

```bash
remote-agent service add --name NAME \
  --command '/root/.local/bin/<cmd> serve --port PORT' \
  --port PORT --port-provider cloudflare_owned \
  --port-base-domain xhd2015.xyz --port-subdomain NAME \
  --start
```

- Domain = `<subdomain>.<base-domain>`; loopback bind is fine (cloudflared is on-host).
- `service add` may block while the tunnel connects — confirm via `service list`
  (`status=connecting` → domain) instead of killing it.

## 4. Verify

```bash
remote-agent service list                               # running + Port row with https domain
curl -fsS https://<subdomain>.xhd2015.xyz/api/.../health
```

Plus: API parity vs local data, root HTML, browser check, `service logs NAME`.

## Ops

| Task | Run |
|------|-----|
| Upgrade binary | `remote-agent install <cmd>` → `remote-agent service restart NAME` (atomic mv; old inode until restart) |
| Refresh data | `exec rm -rf <dest>` → upload → rm process-owned files |
| Rollback | `remote-agent service stop NAME` |

| Wrong | Correct |
|-------|---------|
| Re-upload into existing dir to refresh | `rm -rf` first (cp -R nests) |
| Kill silent `service add` | may block on tunnel connect; check `service list` |
| `server restart` for service binaries | topic **service** (`service upgrade`) |

Real example: ai-workshop (`marcus ai-workshop web --port 8421`) → `ai-workshop-aes8421.xhd2015.xyz`.
