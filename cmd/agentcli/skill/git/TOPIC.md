---
name: remote-agent/git
description: >-
  Clone/fetch/pull/push git repos on the remote machine via remote-agent git.
---

# Git

Operate repositories **on the remote** (not the Mac worktree).

```bash
remote-agent git clone https://github.com/example/project.git
remote-agent git clone https://github.com/example/private.git ~/project --git-token TOKEN
remote-agent git -C ~/project fetch
remote-agent git -C ~/project pull
remote-agent git -C ~/project push
```

For sealed GitHub SSH keys packed into the sandbox, prefer **remote-agent-manager devbox**
`run` with the SSH primary rather than ad-hoc host keys.
Do not paste tokens into skill docs or logs.
