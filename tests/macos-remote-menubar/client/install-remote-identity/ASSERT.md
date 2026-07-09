## Expected

1. `InstallAppName` is `ai-critic-remote-macos`.
2. `InstallBundleID` is `com.xhd2015.ai-critic-remote-macos`.
3. `EmbedsServerBinary` is `false`.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing install-remote.sh; wrong names; embedding `ai-critic` server into remote app.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.InstallAppName != "ai-critic-remote-macos" {
		t.Fatalf("InstallAppName = %q, want ai-critic-remote-macos (sources: %v)",
			resp.InstallAppName, resp.SourcesChecked)
	}
	if resp.InstallBundleID != "com.xhd2015.ai-critic-remote-macos" {
		t.Fatalf("InstallBundleID = %q, want com.xhd2015.ai-critic-remote-macos (sources: %v)",
			resp.InstallBundleID, resp.SourcesChecked)
	}
	if resp.EmbedsServerBinary {
		t.Fatalf("remote install must not embed server binary (sources: %v)", resp.SourcesChecked)
	}
}
```
