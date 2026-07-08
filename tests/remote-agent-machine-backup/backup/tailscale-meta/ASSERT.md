## Expected Output

Dry-run summary includes `TAILSCALE(.backup/tailscale-config.json):` after ENV and before
TOTAL with VERSION/MODE/SOCKS5 table, DAEMON cmdline, SETUP steps, SHELL HISTORY (bash/zsh),
and PEERS table listing mock peers. Archive contains valid `tailscale-config.json` with
redacted prefs and shell history. Stdout ends with `\n`.

## Expected

1. Exit code 0.
2. Dry-run combined output has TAILSCALE section after ENV, before TOTAL.
3. TAILSCALE section contains DAEMON, SETUP, SHELL HISTORY, PEERS, mock self IP and peer names.
4. Archive lists `.backup/tailscale-config.json`.
5. Archive JSON version `1.0`, `running: true`, prefs redacted, setup history includes tailscale lines.
6. Stdout ends with `\n`.

## Side Effects

Creates `tailscale-meta.tar.xz` under `agentHome`.

## Errors

- Missing TAILSCALE section or archive member when mock is seeded.
- Private keys leaked in archive prefs.

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
	tailscaleCapturedAtRE     = regexp.MustCompile(`captured_at: .+  \(running\)`)
	tailscaleVersionDataRE    = regexp.MustCompile(regexp.QuoteMeta(tailscaleFixtureVersion) + `.*userspace-networking.*` + regexp.QuoteMeta(tailscaleFixtureSocks5) + `.*` + regexp.QuoteMeta(tailscaleFixtureSelfIP) + `.*` + regexp.QuoteMeta(tailscaleFixtureDNSName))
	tailscaleVersionHeaderRE  = regexp.MustCompile(`VERSION.*MODE.*SOCKS5.*TAILSCALE IP.*MAGIC DNS`)
	tailscaleSetupStepRE      = regexp.MustCompile(`(?m)^\s+1\.`)
	tailscaleBashHistoryRE    = regexp.MustCompile(`\[bash\].*tailscale`)
	tailscaleZshHistoryRE     = regexp.MustCompile(`\[zsh\].*tailscale`)
	tailscalePeersHeaderRE    = regexp.MustCompile(`PEERS \(\d+\)`)
	tailscalePeerARowRE       = regexp.MustCompile(regexp.QuoteMeta(tailscaleFixturePeerAName) + `.*` + regexp.QuoteMeta(tailscaleFixturePeerAIP) + `.*linux`)
	tailscalePeerBRowRE       = regexp.MustCompile(regexp.QuoteMeta(tailscaleFixturePeerBName) + `.*` + regexp.QuoteMeta(tailscaleFixturePeerBIP) + `.*macOS`)
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

	tailSection := tailscaleSummarySection(combined)
	if tailSection == "" {
		t.Fatalf("missing TAILSCALE summary section; got:\n%s", combined)
	}
	header := metaSectionHeaderLines(tailSection, 1)
	assert.Output(t, header, `---
version: 2
---
TAILSCALE(.backup/tailscale-config.json):
`)

	if !tailscaleCapturedAtRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing running captured_at subheader; section:\n%s", tailSection)
	}

	if !tailscaleVersionHeaderRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing VERSION/MODE/SOCKS5 header; section:\n%s", tailSection)
	}
	if !tailscaleVersionDataRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing version row with fixture values; section:\n%s", tailSection)
	}
	if !strings.Contains(tailSection, tailscaleFixtureCmdline) {
		t.Fatalf("TAILSCALE section missing daemon cmdline %q; section:\n%s", tailscaleFixtureCmdline, tailSection)
	}
	if !tailscaleSetupStepRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing SETUP steps; section:\n%s", tailSection)
	}
	if !tailscaleBashHistoryRE.MatchString(tailSection) || !tailscaleZshHistoryRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing shell history lines; section:\n%s", tailSection)
	}
	if !tailscalePeersHeaderRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing PEERS header; section:\n%s", tailSection)
	}
	if !tailscalePeerARowRE.MatchString(tailSection) || !tailscalePeerBRowRE.MatchString(tailSection) {
		t.Fatalf("TAILSCALE section missing peer rows; section:\n%s", tailSection)
	}

	for _, needle := range []string{"DAEMON", "SETUP", "SHELL HISTORY", "PEERS", "NAME", "TAILSCALE IP", "OS", "STATUS"} {
		if !strings.Contains(tailSection, needle) {
			t.Fatalf("TAILSCALE section missing %q; section:\n%s", needle, tailSection)
		}
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if !memberListContains(members, ".backup/tailscale-config.json") {
		t.Fatalf("archive missing .backup/tailscale-config.json; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/tailscale-config.json")
	snap := parseTailscaleConfigJSON(t, raw)
	assertTailscaleConfigBasics(t, snap)
	assertTailscalePrefsRedacted(t, snap.Prefs)
	assertTailscaleSetupHistory(t, snap)

	statusStr := string(snap.Status)
	if !strings.Contains(statusStr, `"BackendState": "Running"`) {
		t.Fatalf("tailscale status missing BackendState Running; got:\n%s", statusStr)
	}
	if !strings.Contains(statusStr, tailscaleFixtureSelfIP) {
		t.Fatalf("tailscale status missing self IP %q", tailscaleFixtureSelfIP)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```