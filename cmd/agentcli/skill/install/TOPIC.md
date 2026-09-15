---
name: remote-agent/install
description: >-
  Cross-build a local Go CLI for the remote OS/arch and upload it to the
  remote PATH (~/.local/bin or an existing user copy).
---

# Install

```bash
remote-agent install wrk
remote-agent install wrk --dir ~/src/wrk
remote-agent install wrk --dir ~/src/wrk --dry-run
```

Build detection matches `wrk --reinstall-local`:

| `<cmd>` | Preference |
| --- | --- |
| equals module basename | `./script/<cmd>/install` → `./script/install` → `./cmd/<cmd>` |
| otherwise | `./script/<cmd>/install` → `./cmd/<cmd>` |

Install scripts must copy the binary into `$INSTALL_TO_DIR` and honor
`$INSTALL_GOOS` / `$INSTALL_GOARCH` for the product `go build`. `go run` of
the script stays host-native (setting `GOOS` on `go run` would exec-format
the installer). Empty staging after a successful script is a hard error.

| Step | Detail |
| --- | --- |
| Target | Remote `os_info` / binary name (`--goos` / `--goarch` override) |
| Dest | Existing LookPath under `$HOME`, else `~/.local/bin/<cmd>` |
| System paths | `/usr/bin` etc. are not overwritten; falls back to `~/.local/bin` with `warning:` |
| PATH | When writing `~/.local/bin`, the same rc checker as local `EnsureOnPATH` is applied on the remote |
