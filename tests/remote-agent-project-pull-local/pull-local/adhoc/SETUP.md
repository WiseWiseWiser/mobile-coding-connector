# Scenario

**Feature**: adhoc pull-local with explicit `--mode`

```
remote-agent project pull-local --adhoc <remote-dir> --mode git-fetch|download ...
```

## Preconditions

Remote path is a git repo; may be unregistered.

## Steps

1. Leaf prepares remote (and optional local) repos.
2. Invoke pull-local with `--adhoc` / absolute path and mode flags.

## Context

Grouping for adhoc mode validation and dry-run planning.
