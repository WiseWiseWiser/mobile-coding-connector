---
name: remote-agent/service
description: >-
  remote-agent service lifecycle; prefer service upgrade for binary replace with
  remembered --target.
---

# Service

Long-lived processes on the remote ai-critic host.

```bash
remote-agent service list
remote-agent service add --name NAME --command '…'
remote-agent service start|stop|restart|update|logs NAME
```

## Binary ship: `service upgrade`

Uploads while running → stop → move tmp onto target → start.

```bash
remote-agent service upgrade my-svc ./my-svc-linux --target /usr/local/bin/my-svc
# later (reuses remembered Upgrade: path on list):
remote-agent service upgrade my-svc ./my-svc-linux
```

Upgrade uses client **default compress** (no `--no-compress` on upgrade today).  
Help: `remote-agent service upgrade --help`.

| Wrong | Correct |
|-------|---------|
| `upload` + `exec install` + `restart` for routine binary | **`service upgrade NAME BIN [--target …]`** |
| First upgrade without `--target` (lands in `~/basename`) | **`--target /abs/or/~/path`** once; later omit |
| Expect command wrap to change via upgrade | **`service update --command …`** separate |
| `sandbox.bin -- cmd` (cwd = session root) | `sandbox.bin -- sh -c 'cd WORK && exec cmd'` |
| Swap outer `sandbox-ssh.bin` while held | **Stop → upload → start** (text file busy) |
| `prune` while service holds bins | Stop first, or **remote-devbox** prune rules |
