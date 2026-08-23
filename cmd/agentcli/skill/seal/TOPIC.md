---
name: remote-agent/seal
description: >-
  Refresh sealed SMC tokens via remote-devbox; sync passwd-home /root/.smc when needed.
---

# Seal refresh (SMC tokens)

Happy path (**no service restart** if outer runner has event socks):

```bash
remote-devbox refresh              # Mac: SMC pack + upload + notify-event
# launchd: xyz.xhd2015.remote-seal-refresh (every 24h)
```

Requires remote: `sandbox-ssh.bin` listening under `$KOOL_SANDBOX_ROOT/events/*.sock`
+ `/tmp/kool-events` (or `kool`) with `sandbox notify-event`.

## Pitfall: passwd home vs sandbox `$HOME`

After refresh, packed `$HOME/.smc` can be fresh while host `smc token status`
still **expired** — host `smc` often reads **passwd home** `/root/.smc`
(`/dev/shm` is often `noexec`, so packed `bin/smc` may not run).

```bash
remote-agent exec /tmp/sandbox-ssh.bin -- bash -lc \
  'cp -f "$HOME/.smc/smc_token.json" /root/.smc/smc_token.json'
```

Pack/refresh detail → skill **remote-devbox**.
