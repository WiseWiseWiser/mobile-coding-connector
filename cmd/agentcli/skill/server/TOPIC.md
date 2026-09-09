---
name: remote-agent/server
description: >-
  remote-agent server build-next, upgrade --from-source, and restart.
---

# Server

Manage Server UI actions plus local-source upgrade.

```bash
remote-agent server build-next
remote-agent server build-next --project my-project-id
remote-agent server upgrade --from-source
remote-agent server upgrade --from-source --source-dir ~/src/ai-critic
remote-agent server restart
```

| Command | Notes |
|---------|--------|
| `build-next` | Remote `/api/build/build-next` (builds a registered project on the server) |
| `upgrade --from-source` | Local cross-build → `upload-next` → `restart`; origin-scan when `--source-dir` omitted |
| `restart` | `/api/server/exec-restart` |

Replacing a **managed service** binary (not the ai-critic core) → topic **service**
(`service upgrade`), not `server restart` / `server upgrade`.
