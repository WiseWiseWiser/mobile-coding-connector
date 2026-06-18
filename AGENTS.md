# Verify the changes

```sh
go run ./script/build
```

# Theming (CSS)

## Core lesson

Undefined CSS custom properties fail **silently**. `var(--mcc-surface, #fff)` does not error when `--mcc-surface` is missing — it uses the fallback (`#fff`), which produced bright white cards on an otherwise dark app (File Transfer).

## Rules for v2/mcc UI

1. **Use only tokens from `ai-critic-react/src/v2/mcc/theme.css`** — e.g. `--mcc-bg-card`, `--mcc-text-primary`, `--mcc-border-default`, `--mcc-accent-blue`. Do not invent new `--mcc-*` names.
2. **Do not use light-theme fallbacks** in `var()`. If a token is missing, fix the token reference; a white fallback hides the bug until someone opens the page.
3. **Copy patterns from existing dark views** before writing new CSS — `ToolsView.css`, `UploadFileView.css`, and `ManageServerView.css` are good references.
4. **When adding a new view**, read `theme.css` first, then grep the codebase for how similar elements (cards, buttons, errors, empty states) are styled elsewhere in `v2/mcc/`.
5. **Smoke-check in the running app** — build passing does not catch wrong colors; a quick visual check on one dark-themed page catches undefined-variable fallbacks immediately.

