---
name: remote-agent/edit
description: >-
  Edit a remote file in a local editor: staged under /tmp/remote-agent-edit,
  written back only when the md5 is unchanged (conflict → keep local, resolve).
---

# Edit

```bash
remote-agent edit /etc/nginx/nginx.conf              # $VISUAL/$EDITOR, else vim
remote-agent edit '~/notes/todo.md' --editor=nano
remote-agent edit /etc/app/config.yaml --editor=code --remember-flags
```

| Behavior | Detail |
|----------|--------|
| Staging | Remote file → `/tmp/remote-agent-edit/<remote-path>` (dirs/files `0700`/`0600`); `--work-dir` overrides |
| Missing remote file | Staged copy starts empty; saving creates the file |
| Write-back | Uploaded with the md5 recorded at download time as precondition |
| No change | Editor exit without content change → `file not changed`, no request |
| Conflict | Remote md5 differs → save refused, nothing written, staged copy kept |
| Editor exit ≠ 0 | Nothing uploaded (abort path, e.g. vim `:cq`) |
| GUI editors | `code`/`code-insiders`/`codium` get `--wait`; `subl`/`mate`/`atom` get `-w` |
| Terminal editors | `vim`/`nano`/… need a tty on stdin; pass a GUI editor for non-interactive runs |
| `--remember-flags` | Saves `--editor`/`--work-dir` to `~/.ai-critic/remote-agent-config.json` (`remembered_flags.edit`) for later runs; no flags given clears it |
| Symlinks | Writes through a symlink (target replaced, link preserved) |
| Mode | Existing file mode preserved; new files `0644` |

## Resolving a conflict

The remote changed while the staged copy was open. Reconcile, then push:

```bash
remote-agent download /etc/app/config.yaml /tmp/remote-agent-edit/etc/app/config.yaml.remote
# merge into /tmp/remote-agent-edit/etc/app/config.yaml
remote-agent upload /tmp/remote-agent-edit/etc/app/config.yaml /etc/app/config.yaml
```

Re-running `edit` re-downloads the remote content and replaces the staged copy,
so merge before re-running.

Server side: `POST /api/files/write` (`path`, `expected_md5`, `file`) returns
409 with both hashes when the precondition fails; a missing file counts as the
md5 of empty content.
