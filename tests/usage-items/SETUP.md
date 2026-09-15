# Scenario

**Feature**: usage item registry, provider rendering, and the usage item HTTP API

```
registry file <- usageitems.Service (fetch workers) -> GET /api/usage/items
CLI (local-agent usage ...) -> POST /api/usage/items/{add,update,remove,default}
```

## Preconditions

1. `macosapp/usageitems` exposes `NewStore`, `NewService`, `Registry`, `Item`,
   `ItemView`, `ListResponse`, `AddRequest`, `UpdateRequest`, `RemoveResult`,
   and `TestExported_SetSnapshotFetcher`.
2. `server/usage.RegisterAPI` mounts the handlers; `TestExported_SetItemsService`
   points them at a service backed by a temporary registry file.
3. Provider fetching is replaced by fixtures: any item whose `Home` equals
   `FailHome` fails, every other item reports a ready snapshot.
4. No subprocess, no network, no real provider credentials.

## Steps

1. Leaf `Setup` picks an `Op` and seeds the registry contents.
2. Root `Run` writes the seed registry, builds the service, runs the op, and
   records views, warnings, and errors.
3. Leaf `Assert` checks the rendered menu text, registry state, or HTTP status.

## Context

The menu bar is a thin renderer: the server decides the title and dropdown
strings, the CLI edits the registry. These leaves lock that contract so the
macOS app never has to format usage text itself. The harness lives in
`DOCTEST.md`.
