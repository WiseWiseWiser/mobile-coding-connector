# Signed protocol: Codex “Update available” modal → Skip → continue

**Status:** CAPTURED and VERIFIED on live Codex CLI  
**Date:** 2026-07-10  
**Codex version:** `codex-cli 0.143.0` (prompted update to `0.144.0`)  
**Capture method:** `tty-watch` ephemeral session + raw `InjectInput` (Go)  
**Argv (production usage fetch):**  
`codex --dangerously-bypass-approvals-and-sandbox -c mcp_servers={}`

This directory freezes observed TUI snapshots and the key sequence that
successfully chooses **Skip** and continues into a normal session so `/status`
can be scraped. Product code must follow this protocol; do not invent keys.

---

## Theory under test

| Claim | Result |
|-------|--------|
| Usage fetch blocks on update **menu** modal (not network) | **CONFIRMED** (`01`) |
| Default selection is **1. Update now** | **CONFIRMED** (`01`) |
| One CSI **Down** (`\x1b[B`) moves cursor to **2. Skip** | **CONFIRMED** (`02`) |
| Must **verify** `›` on Skip before Enter | **REQUIRED** (Enter alone would update) |
| Enter (`\r`) after Skip dismisses the **menu** | **CONFIRMED** (`03b`) |
| Residual non-menu **banner** may still say “Update available” | **CONFIRMED** (`03b`, `04`) |
| After dismiss, `/status` yields parseable usage fields | **CONFIRMED** (`05`) |

---

## Canonical key sequence (signed)

```
1. Detect blocking modal
2. Inject CSI Down once:  hex 1b5b42   (= ESC [ B)
3. Snapshot + assert selection on Skip
4. Inject Enter:         hex 0d        (= \r)
5. Poll until menu options gone (see predicates)
6. Continue existing wait-for-idle + /status flow
```

### Keys that failed (do not use as primary)

First shell-based attempt with `tty-watch send` did not move selection (likely
argument/escaping loss for ESC). Go `InjectInput` with raw bytes works.

Probed and **not needed** once CSI Down worked (session was not exhausted for
later probes after early success): digit `2`, Tab, `j`, SS3 Down, Ctrl-N.

---

## Fixtures (SHA-256)

| File | SHA-256 | Role |
|------|---------|------|
| `01-update-modal-default.snapshot.txt` | `5e67bc8cc2aac0970e7695a57c1689a52171756895dc2138f9c4828e5f1e9b57` | Default menu; `›` on Update now |
| `02a-csi-down-x1.snapshot.txt` | `b26b19ecdde37e73e9e9023bf4b2c004bfd776c47541c748fa634a352744dff2` | After one CSI Down |
| `02-skip-selected.snapshot.txt` | `b26b19ecdde37e73e9e9023bf4b2c004bfd776c47541c748fa634a352744dff2` | Canonical Skip-selected (= `02a`) |
| `03-after-enter.snapshot.txt` | `b26b19ecdde37e73e9e9023bf4b2c004bfd776c47541c748fa634a352744dff2` | Immediate post-Enter (~800ms); **still menu** (lag) |
| `03b-menu-dismissed.snapshot.txt` | `11c3c357fab3b6dd1976e26ed94a4a884c77052faeb81f21d5a03f9e7928ed2c` | First post-Enter frame with **menu options gone** |
| `04-idle-prompt.snapshot.txt` | `69c26262b22f192aaf51cd3cb680ccb345db0ccaea03c1351b8df0f8aeb5f76c` | Main TUI + optional update **banner** + prompt |
| `05-status-fields.snapshot.txt` | `63ee40ae79a6adfce34c3950e78559a5087aeb41cb9363215878d46e7732c772` | `/status` usage fields visible |
| `RESULT.json` | machine-readable win summary | |

Re-hash after any fixture edit:

```sh
cd tests/codex-usage/testdata/update-modal-skip && shasum -a 256 *.txt RESULT.json
```

---

## Step predicates (machine-checkable)

### STEP `detect_modal` ← `01`

**Assert all:**

1. Case-insensitive contains `update available`
2. Contains menu options: `1. Update now`, `2. Skip`, `3. Skip until next version`
3. Contains `Press enter to continue` (case-insensitive OK)
4. Selection line matches: `›` (or U+203A) + `1.` + `Update`
5. No selection line matching `›` + `2.` + `Skip` (without “until”)

**Action if true:** go to `select_skip` (do **not** send `/status`).

### STEP `select_skip`

**Input:** exactly one CSI Down: bytes `1b 5b 42`

**Fixture:** `02-skip-selected.snapshot.txt` / `02a-csi-down-x1.snapshot.txt`

**Assert all before Enter:**

1. Still a blocking menu (`2. Skip` and `Skip until next version` present)
2. Selection line matches: `›` + `2.` + `Skip` and does **not** contain `until`
3. No selection on `1. Update now`

**If assert fails:** retry Down at most once more; if still fail → **error**  
`could not select Skip on update prompt` — **never** send Enter.

### STEP `confirm_skip`

**Input:** Enter: byte `0d` (`\r`) — only if `select_skip` passed.

**Do not** treat immediate next frame as success (`03` may still show the menu).

**Poll until menu dismissed** (predicate for `03b` / success):

1. Does **not** contain `Skip until next version`
2. Does **not** contain a list line `2. Skip` (menu option)
3. Does **not** contain `Press enter to continue` as the modal footer  
   (banner may still mention updates)

**Note:** A top **banner** may still contain `Update available!` and  
`Run npm install -g @openai/codex to update.` — that is **not** the blocking menu.

### STEP `continue_status` ← `04` → `05`

1. Wait until main prompt is idle (existing `checkCodexWritable` / prompt logic),
   **after** fixing banner vs menu detection (see below).
2. Send `/status\n\r` (production bytes).
3. Wait until parseable usage:
   - `Monthly credit limit:` … `% left`
   - `N of M credits used` (commas optional)
   - `(resets …)`

`05` is an example of a successful `/status` screen (fields may vary by account).

---

## Critical product implication (beyond keys)

Today `checkCodexWritable` treats **any** screen containing `update available` as
`State: "loading"`. After Skip, the **info banner** still contains that phrase
(`03b` / `04`), so wait-for-prompt would **still hang** even with correct Skip
keys unless detection is narrowed to the **blocking menu**, e.g.:

| Screen | Signals | Treat as |
|--------|---------|----------|
| Blocking modal | `update available` + (`skip until next version` \| `press enter to continue` \| numbered `1. Update` / `2. Skip`) | loading + auto-Skip protocol |
| Residual banner only | `update available` + `Run npm install` **without** menu options | **not** blocking; allow idle when `›` prompt ready (still respect `model:loading`) |

---

## Re-capture / re-sign

When Codex UI changes, re-run a capture that:

1. Starts production argv under isolated `TTY_WATCH_HOME`
2. Injects raw bytes via `InjectInput` (not shell-escaped `tty-watch send` for ESC)
3. Overwrites fixtures only if predicates still hold
4. Updates SHA-256 table in this file and `RESULT.json`

Live probe result (this capture):

```json
{
  "codex_version": "0.143.0",
  "selected_fixture": "02a-csi-down-x1.snapshot.txt",
  "success": true,
  "winning_keys_hex": "1b5b42"
}
```

---

## Non-goals of this protocol

- Auto-select “Update now”
- Prefer “Skip until next version” (option 3) — product choice is **Skip** (option 2)
- Rely on digit `2` without re-proving on a clean session
- Treat residual banner as failure after menu dismiss
