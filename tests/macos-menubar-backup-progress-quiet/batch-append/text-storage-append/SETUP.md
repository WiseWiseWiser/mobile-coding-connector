# Scenario

**Feature**: flush writes via textStorage.append (cheap batch), not sole string +=

```
# required
flush -> textView.textStorage?.append(NSAttributedString…)  # or equivalent

# banned as sole hot path
appendOnMain -> textView.string += line + "\n"  # full rewrite thrash
```

## Preconditions

`BackupProgressWindow.swift` uses `textStorage` + `append` (or replace on storage).
Per-line-only `string +=` without textStorage is insufficient.

## Steps

1. ClientLeaf=text-storage-append.

## Context

REQUIREMENT #6; RED on current `textView.string += line + "\n"`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "text-storage-append"
	return nil
}
```
