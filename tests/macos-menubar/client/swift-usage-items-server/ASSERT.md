## Expected

1. `UsageViaServerClient` is `true` — `ServerClient` requests `/api/usage/items`
   from the server port.
2. `AppUsesServerUsageItems` is `true` — `AppState.refresh` calls
   `ServerClient.shared.usageItems()`.
3. `UsageViaDaemonClient` is `false` — no Swift source under `macos-ai-critic/`
   calls `grokUsage` / `codexUsage` any more; `LegacyUsageCallers` lists any
   offender and must be empty.

## Side Effects

- None (read-only source inspection).

## Errors

- Menu-bar usage still comes from the retired per-provider client helpers, or
  from the daemon port instead of the server usage-items registry.

```go
import (
"testing"

"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
if err != nil {
t.Fatal(err)
}
if !resp.UsageViaServerClient {
t.Fatalf("usage items not fetched from ServerClient (sources: %v)", resp.SwiftSourcesChecked)
}
if !resp.AppUsesServerUsageItems {
t.Fatalf("AppState.refresh does not call ServerClient.shared.usageItems() (sources: %v)", resp.SwiftSourcesChecked)
}
if resp.UsageViaDaemonClient {
t.Fatalf("Swift sources still call grokUsage/codexUsage: %v (sources: %v)", resp.LegacyUsageCallers, resp.SwiftSourcesChecked)
}
}
```
