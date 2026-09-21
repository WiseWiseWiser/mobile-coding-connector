---
name: remote-agent/exec
description: >-
  Run one-shot remote shell via remote-agent exec (verbatim argv, exit mirrors remote).
---

# Exec

Args after `exec` go to the remote **verbatim**; exit code mirrors the remote process.

```bash
remote-agent exec ls -la /tmp
remote-agent exec sh -c 'echo hi; uname -a'
remote-agent exec bash -c 'curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:PORT/health'
```

| Use | Command |
|-----|---------|
| Bare host shell | `remote-agent exec …` |
| Packed SMC / git SSH | **remote-agent-manager devbox** `run` (not bare `exec smc` expecting sealed HOME) |

Interactive TTY attaches when stdin/stdout are a terminal (PTY mode).
