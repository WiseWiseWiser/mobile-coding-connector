---
name: remote-agent/service
description: >-
  remote-agent service lifecycle; prefer service upgrade for binary replace with
  remembered --target, and for configured build/swap steps.
---

# Service

Long-lived processes on the remote ai-critic host.

```bash
remote-agent service list
remote-agent service add --name NAME --command '…'
remote-agent service start|stop|restart|update|logs NAME
remote-agent service upgrade NAME [BIN]
```

## Upgrade: binary and/or configured steps

One pipeline: **pre-stop steps (service live) → stop → move uploaded binary →
post-stop steps (service stopped) → start.**

```bash
# Binary ship: uploads while running, then swaps after the stop.
remote-agent service upgrade my-svc ./my-svc-linux --target /usr/local/bin/my-svc
remote-agent service upgrade my-svc ./my-svc-linux   # reuses remembered target

# Steps only, configured on the definition (repeatable, ordered).
remote-agent service add --name kool --command 'node server.js' \
  --working-dir /root/kool \
  --upgrade-pre-stop-cmd 'git fetch --all' \
  --upgrade-pre-stop-cmd 'go build -o /tmp/kool.new ./cmd/kool' \
  --upgrade-post-stop-cmd 'mv /tmp/kool.new "$REMOTE_AGENT_UPGRADE_TARGET"' \
  --upgrade-target /root/.local/bin/kool
remote-agent service upgrade kool

# One-run override (does not change the definition).
remote-agent service upgrade kool --upgrade-pre-stop-cmd 'go build ./...'
```

Every step runs through `bash -lc 'set -eo pipefail; <cmd>'` in the service
working dir and env, so a failure anywhere in a pipeline aborts the step.
`$REMOTE_AGENT_UPGRADE_TARGET` carries the resolved target into each step.
Step output streams to the terminal and is appended to the service log.
Default per-step timeout is 15m; `service update --upgrade-timeout 0` disables.

| Failure | Result |
|---------|--------|
| Pre-stop step | Aborts; service **never stopped**, still serving the old build |
| Post-stop step or binary move | Aborts, **service restarted**, then reports |
| Restart during recovery | Reports that the service is **stopped** |

Because pre-stop runs while the service is live, build to a temp path: writing
over a running binary fails with `text file busy`. Swap it in post-stop.

## Worked example: rebuild a service from its source checkout

A Node/Go service whose source lives in a remote checkout upgrades by pulling
and rebuilding in place, then restarting into the new build:

```bash
remote-agent service update dsh \
  --upgrade-pre-stop-cmd 'export PATH=/opt/node24/bin:$PATH; cd /opt/dsh && git fetch origin BRANCH && git reset --hard FETCH_HEAD' \
  --upgrade-pre-stop-cmd 'export PATH=/opt/node24/bin:$PATH; cd /opt/dsh && pnpm install --frozen-lockfile' \
  --upgrade-pre-stop-cmd 'export PATH=/opt/node24/bin:$PATH; cd /opt/dsh && pnpm run build' \
  --upgrade-post-stop-cmd 'export PATH=/opt/node24/bin:$PATH; /opt/node24/bin/node /opt/dsh/apps/cli/lib/bin.js --version' \
  --upgrade-timeout 30m

remote-agent service upgrade dsh
```

Rules this example encodes:

| Rule | Why |
|------|-----|
| Fetch the **branch the service was deployed from** | The checkout may sit on a feature branch; resetting to `origin/master` silently downgrades it |
| `git fetch` + `reset --hard FETCH_HEAD`, never `git clean -fdx` | `-x` deletes `node_modules` (multi-GB reinstall) |
| `export PATH=<pinned runtime>:$PATH` in **every** step | Steps are separate shells; the ambient toolchain may be too old (e.g. `pnpm` 11 needs node ≥ 22 while ambient is node 16) |
| `pnpm install --frozen-lockfile` | Fails loudly when the lockfile and `package.json` disagree |
| Post-stop `--version` check | Turns a broken build into a loud failure instead of a crash-loop; it runs while the service is stopped |
| `--upgrade-timeout 30m` | A cold install plus a full monorepo build exceeds the 15m default |

Verify an upgrade with `remote-agent exec sh -c 'cd /opt/dsh && git rev-parse --short HEAD'`
and compare against the local `git rev-parse HEAD`; a first run against the
already-deployed commit is a safe end-to-end rehearsal. Rebuilding in place
leaves the service live throughout the build, but the tree it serves from
changes under it — for a service whose web assets are read per request, keep a
staging checkout and `mv` it into place in a post-stop step instead.

| Wrong | Correct |
|-------|---------|
| `go build -o /path/to/running-binary` in a pre-stop step | Build to **`/tmp/x.new`**, swap in post-stop |
| `service restart` after a source change | **`service upgrade NAME`** (steps) |
| First upgrade without `--target` (lands in `~/basename`) | **`--target /abs/or/~/path`** once; later omit |
| Expect command wrap to change via upgrade | **`service update --command …`** separate |
| `upload` + `exec install` + `restart` for routine binary | **`service upgrade NAME BIN [--target …]`** |
| `sandbox.bin -- cmd` (cwd = session root) | `sandbox.bin -- sh -c 'cd WORK && exec cmd'` |
| Swap outer `sandbox-ssh.bin` while held | **Stop → upload → start** (text file busy) |
| `prune` while service holds bins | Stop first, or **remote-agent-manager devbox** prune rules |

Help: `remote-agent service upgrade --help`.
