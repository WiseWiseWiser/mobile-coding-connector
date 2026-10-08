---
name: remote-agent/server
description: >-
  remote-agent server build-next, upgrade --from-source, and restart (Manage
  Server page equivalents).
---

# Server

Same actions as the Manage Server UI, streamed over HTTP.

```bash
remote-agent server build-next
remote-agent server build-next --project my-project-id
remote-agent server upgrade --from-source
remote-agent server upgrade --from-source --source-dir ~/src/ai-critic
remote-agent server restart
```

| Command | Stream |
|---------|--------|
| `build-next` | `/api/build/build-next` |
| `upgrade --from-source` | Local cross-build → `upload-next` → restart; origin-scan when `--source-dir` omitted |
| `restart` | `/api/server/exec-restart` |

Replacing a **managed service** binary (not the ai-critic core) → topic **service**
(`service upgrade`), not `server restart` / `server upgrade`.

## Flaky link: split the upgrade

`upgrade --from-source` resumes only within one run (websocket drops); a run that
dies at `upload failed at offset … i/o timeout` restarts the transfer from zero
on retry. A failed upload never touches the running server, so it is safe to
redo — but past a certain size, prefer the direct SSH path:

```bash
scp -C ai-critic-server-linux-amd64 x.dev:/root/servers/ai-critic/ai-critic-server-v31.tmp
ssh x.dev 'cd /root/servers/ai-critic && chmod 755 ai-critic-server-v31.tmp && mv ai-critic-server-v31.tmp ai-critic-server-v31 && md5sum ai-critic-server-v31'
remote-agent server restart        # execs the highest ai-critic-server-vN, PID unchanged
```

Compare that `md5sum` with the local artifact before restarting. Keep the `-vN`
name shape — `restart` finds the newest by that suffix, and older versions stay
on disk for rollback.
