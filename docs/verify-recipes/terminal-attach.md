# Verify recipe — `remote-agent terminal attach`

Autonomous gate before claiming attach work is done. Complements L2 doctests
(`tests/terminal-cli-attach`). This recipe is the human journey.

## Surface / depth

- Surface: CLI + live server + session lifecycle (real PTY)
- Depth: **scenario**
- Mode: sandbox (isolated HOME / `AI_CRITIC_HOME` / `TTY_WATCH_HOME`)

## Change-scoped binaries

- `ai-critic-server` (`go build -o … .`) — first-frame lives here
- `remote-agent` (`go build -o … ./cmd/remote-agent`) — attach handshake / exited gate
- `tty-watch` on PATH

Rebuilding only the CLI is **not** enough for this recipe.

## Run

```sh
./script/verify-terminal-attach-e2e.sh
```

Exit 0 = may say done. Non-zero = not done. Do not ask the user to reproduce.

## Scenarios (must all pass)

| # | Journey | Fail if |
|---|---------|---------|
| 1 | `terminal new` | no prompt in `tty-watch snapshot` |
| 2 | `terminal attach <id>` of that **running** session | snapshot timeout / empty / no prompt (the hang) |
| 3 | type `echo VERIFY_ATTACH_MARKER` on the attach TTY | marker missing |
| 4 | attach first snapshot contains `\e[?1049l`, `\e[H\e[2J`, or `\e[2J` | Ctrl-L refresh |
| 5 | Ctrl-] on the attach TTY | no `detached` line; list not `running`; re-attach fails |
| 6 | stop the `new` writer; list is `exited`; `terminal attach` | no `is exited` error within 3s (mute hang) |

## Not sufficient

- L2 doctest GREEN alone
- `go test ./cmd/agentcli`
- Rebuilding `remote-agent` without a server built from the same tree
