## Expected Output

Dry-run summary after EXCLUDED prints meta sections in order: GIT REPOS (none when
no repos), `INSTALLED SOFTWARE(.backup/installed.json):` with `captured_at` subheader
and NAME/VERSION/PATH table header, then `ENV(.backup/ENV):` with indented `KEY=VALUE`
lines, optionally `TAILSCALE(.backup/tailscale-config.json):` when running (this leaf:
absent), optionally `CLOUDFLARED(.backup/cloudflared-config.json):` when running (this
leaf: absent), optionally `SYSTEMD SERVICES(.backup/systemd-services.json):` when
systemctl available (this leaf: absent), then `TOTAL`. Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. `GIT REPOS(.backup/git-repo-worktrees.json): (none)` appears before INSTALLED.
4. INSTALLED SOFTWARE section has `captured_at` and NAME/VERSION/PATH table headers.
5. ENV section has at least one `KEY=VALUE` line (4-space indent in summary).
6. Meta sections appear before `TOTAL` in order GIT REPOS → INSTALLED → ENV → [TAILSCALE?] → [CLOUDFLARED?] → [SYSTEMD SERVICES?] → TOTAL.
7. Combined output does not contain `CLOUDFLARED(.backup/cloudflared-config.json):` (no mock).
8. Combined output does not contain `SYSTEMD SERVICES(.backup/systemd-services.json):` (no mock).
9. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing INSTALLED or ENV summary sections.
- Meta sections after TOTAL or out of order.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var envSummaryLineRE = regexp.MustCompile(`(?m)^\s{4}\w+=`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Combined, "dry-run: machine backup plan") {
		t.Fatalf("missing backup plan summary; got:\n%s", resp.Combined)
	}

	assertMetaSectionsBeforeTotal(t, resp.Combined)

	gitSection := gitReposSummarySection(resp.Combined)
	gitHeader := metaSectionHeaderLines(gitSection, 1)
	assert.Output(t, gitHeader, `---
version: 2
---
GIT REPOS(.backup/git-repo-worktrees.json): (none)
`)

	installedSection := installedSummarySection(resp.Combined)
	if installedSection == "" {
		t.Fatalf("missing INSTALLED SOFTWARE summary section; got:\n%s", resp.Combined)
	}
	installedHeader := metaSectionHeaderLines(installedSection, 1)
	assert.Output(t, installedHeader, `---
version: 2
---
INSTALLED SOFTWARE(.backup/installed.json):
`)
	if !regexp.MustCompile(`captured_at: .+  \(\d+ tools\)`).MatchString(installedSection) {
		t.Fatalf("INSTALLED SOFTWARE missing captured_at tools subheader; section:\n%s", installedSection)
	}

	assertInstalledTableHeaders(t, installedSection)
	if !strings.Contains(installedSection, "captured_at:") {
		t.Fatalf("INSTALLED SOFTWARE missing captured_at subheader; section:\n%s", installedSection)
	}

	envSection := envSummarySection(resp.Combined)
	if envSection == "" {
		t.Fatalf("missing ENV summary section; got:\n%s", resp.Combined)
	}
	envHeader := metaSectionHeaderLines(envSection, 1)
	assert.Output(t, envHeader, `---
version: 2
---
ENV(.backup/ENV):
`)

	if !envSummaryLineRE.MatchString(envSection) {
		t.Fatalf("ENV section missing indented KEY=VALUE line; section:\n%s", envSection)
	}

	if strings.Contains(resp.Combined, "CLOUDFLARED(.backup/cloudflared-config.json):") {
		t.Fatalf("unexpected CLOUDFLARED section without mock; got:\n%s", resp.Combined)
	}
	if cloudflaredSummarySection(resp.Combined) != "" {
		t.Fatal("cloudflaredSummarySection non-empty without mock")
	}

	if strings.Contains(resp.Combined, "SYSTEMD SERVICES(.backup/systemd-services.json):") {
		t.Fatalf("unexpected SYSTEMD SERVICES section without mock; got:\n%s", resp.Combined)
	}
	if systemdServicesSummarySection(resp.Combined) != "" {
		t.Fatal("systemdServicesSummarySection non-empty without mock")
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```