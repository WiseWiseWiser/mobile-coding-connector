---
name: remote-agent/cron
description: >-
  Schedule remote shell on the ai-critic host (server-side only; not Mac upload).
---

# Cron

Server-side only (`bash -lc`). Schedules: `--every` \| `--cron` \| `--cron-utc`.

```bash
remote-agent cron list
remote-agent cron add --name NAME --command '…' --every 24h --timeout 5m
remote-agent cron run NAME && remote-agent cron logs NAME
```

| Wrong | Correct |
|-------|---------|
| Cron packs Mac SMC and uploads | Mac **`remote-devbox refresh`**; cron only runs remote commands |
| Expect Mac `upload` from cron | Impossible — schedule is remote-only |

Optional remote process watchdog (generic) is fine; do not confuse with CodeLens
watchdog knowledge.
