# Scenario

**Feature**: autonomous tty-watch attach gate

## Preconditions

- `tty-watch` on PATH (skip otherwise).
- Isolated homes created by `script/verify-terminal-attach-e2e.sh`.

## Steps

1. Leaf sets `req.Phase = "tty-watch-journey"`.
2. Harness execs the script from `d.DOCTEST_ROOT`.
