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
```

| Behavior | Detail |
|----------|--------|
| Default | Whole-file **gzip** when smaller; server gunzips on complete |
| `--no-compress` | Raw chunks (old server without gunzip, or force raw) |
| Resume | Same content → same `file_hash` → skipped cached chunks after interrupt |
| Service binaries | Prefer **`service upgrade`** (see topic **service**), not upload + `mv` + restart |

Atomic replace (non-service): upload `PATH.new` →  
`exec sh -c 'mv PATH.new PATH && chmod +x PATH'`.
