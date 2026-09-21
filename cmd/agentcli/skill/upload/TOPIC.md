---
name: remote-agent/upload
description: >-
  Mac→remote file upload: default whole-file gzip, --no-compress, hash resume, +x.
---

# Upload

```bash
remote-agent upload ./bin /tmp/bin                 # +x when local is executable
remote-agent upload ./x /tmp/x.new                 # stage, then atomic mv
remote-agent upload --no-compress ./bin /tmp/bin   # force raw bytes
remote-agent upload ./srcdir /tmp/apps             # dir: tar.xz → upload → remote apply
remote-agent upload --no-override ./srcdir /tmp/apps
```

| Behavior | Detail |
|----------|--------|
| Default (file) | Whole-file **gzip** when smaller; server gunzips on complete |
| `--no-compress` | Raw chunks for **files** (dir archives are always xz) |
| Directory | Local **tar.xz** pack → one chunked upload → remote extract/merge (`cp -R` dest rules) |
| Dir progress | stderr `[n/4]` stages (`resolve`/`pack`/`upload`/`apply`); stdout product `uploaded …` |
| `--no-override` | Dir only: preflight before tar; refuse if any remote file would be overwritten |
| Resume | Same content → same `file_hash` → skipped cached chunks after interrupt |
| Service binaries | Prefer **`service upgrade`** (see topic **service**), not upload + `mv` + restart |

Atomic replace (non-service): upload `PATH.new` →  
`exec sh -c 'mv PATH.new PATH && chmod +x PATH'`.
