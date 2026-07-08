## Expected Output

Dry-run summary includes `CLOUDFLARED(.backup/cloudflared-config.json):` after ENV and before
TOTAL with VERSION/MODE/TARGET table, DAEMON cmdline, CONFIG path (present, redacted), and
SHELL HISTORY (bash) listing mock quick-tunnel line. Archive contains valid
`cloudflared-config.json` with redacted config and shell history. Stdout ends with `\n`.

## Expected

1. Exit code 0.
2. Dry-run combined output has CLOUDFLARED section after ENV, before TOTAL.
3. CLOUDFLARED section contains DAEMON, CONFIG, SHELL HISTORY, fixture version and target URL.
4. Archive lists `.backup/cloudflared-config.json`.
5. Archive JSON version `1.0`, `running: true`, config redacted, setup history includes cloudflared line.
6. Stdout ends with `\n`.

## Side Effects

Creates `cloudflared-meta.tar.xz` under `agentHome`.

## Errors

- Missing CLOUDFLARED section or archive member when mock is seeded.
- Tunnel/credentials secrets leaked in archive config.

## Exit Code

0.

```go
import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	cloudflaredCapturedAtRE    = regexp.MustCompile(`captured_at: .+  \(running\)`)
	cloudflaredVersionDataRE   = regexp.MustCompile(regexp.QuoteMeta(cloudflaredFixtureVersion) + `.*quick-tunnel.*` + regexp.QuoteMeta(cloudflaredFixtureURL))
	cloudflaredConfigPathRE    = regexp.MustCompile(`\.cloudflared/config\.yml  \(present, redacted\)`)
	cloudflaredShellHistoryRE  = regexp.MustCompile(`\[bash\].*cloudflared`)
	cloudflaredVersionHeaderRE = regexp.MustCompile(`VERSION.*MODE.*TARGET`)
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.BackupPath == "" {
		t.Fatal("BackupPath empty")
	}
	if _, err := os.Stat(resp.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	combined := resp.DryRunCombined
	if combined == "" {
		combined = resp.Combined
	}
	assertMetaSectionsBeforeTotal(t, combined)

	cloudSection := cloudflaredSummarySection(combined)
	if cloudSection == "" {
		t.Fatalf("missing CLOUDFLARED summary section; got:\n%s", combined)
	}
	header := metaSectionHeaderLines(cloudSection, 1)
	assert.Output(t, header, `---
version: 2
---
CLOUDFLARED(.backup/cloudflared-config.json):
`)

	if !cloudflaredCapturedAtRE.MatchString(cloudSection) {
		t.Fatalf("CLOUDFLARED section missing running captured_at subheader; section:\n%s", cloudSection)
	}

	if !cloudflaredVersionHeaderRE.MatchString(cloudSection) {
		t.Fatalf("CLOUDFLARED section missing VERSION/MODE/TARGET header; section:\n%s", cloudSection)
	}
	if !cloudflaredVersionDataRE.MatchString(cloudSection) {
		t.Fatalf("CLOUDFLARED section missing version row with fixture values; section:\n%s", cloudSection)
	}
	if !strings.Contains(cloudSection, cloudflaredFixtureCmdline) {
		t.Fatalf("CLOUDFLARED section missing daemon cmdline %q; section:\n%s", cloudflaredFixtureCmdline, cloudSection)
	}
	if !cloudflaredConfigPathRE.MatchString(cloudSection) {
		t.Fatalf("CLOUDFLARED section missing redacted config path; section:\n%s", cloudSection)
	}
	if !cloudflaredShellHistoryRE.MatchString(cloudSection) {
		t.Fatalf("CLOUDFLARED section missing bash shell history; section:\n%s", cloudSection)
	}

	for _, needle := range []string{"DAEMON", "CONFIG", "SHELL HISTORY", cloudflaredFixtureURL} {
		if !strings.Contains(cloudSection, needle) {
			t.Fatalf("CLOUDFLARED section missing %q; section:\n%s", needle, cloudSection)
		}
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if !memberListContains(members, ".backup/cloudflared-config.json") {
		t.Fatalf("archive missing .backup/cloudflared-config.json; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/cloudflared-config.json")
	snap := parseCloudflaredConfigJSON(t, raw)
	assertCloudflaredConfigBasics(t, snap)
	assertCloudflaredConfigRedacted(t, snap)
	assertCloudflaredSetupHistory(t, snap)
	if snap.Tunnels.Available {
		t.Fatal("cloudflared tunnels.available = true, want false without credentials")
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```