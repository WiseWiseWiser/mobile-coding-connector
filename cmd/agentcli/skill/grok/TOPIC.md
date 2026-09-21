---
name: remote-agent/grok
description: >-
  Start or resume a remote Grok session attached to this terminal tab
  (grok, grok --resume, grok --cwd). Not agent-run.
---

# remote-agent grok

Start or resume a **remote Grok** session and attach **this terminal tab** to its PTY (no new window/tab). Not agent-run.

```bash
remote-agent grok
remote-agent grok --resume 01a09d34-da75-7e11-a815-f06f6dd39bbc
remote-agent grok --cwd /root/seatalk-local-bot --resume 01a09d34-…
```

| Flag | Behavior |
|------|----------|
| (none) | `POST` terminal session with command `grok`, then attach |
| `--resume ID` | command `grok --resume ID`; cwd from `~/.grok/sessions/<urlencoded-cwd>/<ID>/` when found |
| `--cwd DIR` | Override remote working directory |

Detach: **Ctrl-]** (remote Grok keeps running). Re-attach: `remote-agent terminal attach <id-or-name>` (session name `grok` / `grok-<short-id>`).
