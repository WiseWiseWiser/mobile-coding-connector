## Expected Output

Dry-run summary lists `.wrk-test/main` in a GIT REPOS table row with branch `main`,
7-char short sha, `clean` status, commit subject, and ORIGIN URL
`https://github.com/example/backup-fixture.git`. Real backup archive
`.backup/git-repo-worktrees.json` includes `origin_url` with the same URL. Stdout
ends with a trailing newline.

## Expected

1. Exit code 0.
2. Dry-run GIT REPOS table lists main repo metadata and ORIGIN URL column.
3. Archive contains `.backup/git-repo-worktrees.json` with `origin_url` for `.wrk-test/main`.
4. Stdout ends with `\n`.

## Side Effects

Creates `git-repos-origin-url.tar.xz` under `agentHome`.

## Errors

- Missing ORIGIN URL in table row or missing/empty `origin_url` in archive JSON.

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
	gitReposCapturedAtRE = regexp.MustCompile(`captured_at: .+  \(1 repo, 0 worktree\)`)
	gitReposOriginRowRE  = regexp.MustCompile(`repo\s+\.wrk-test/main\s+main\s+[0-9a-f]{7}\s+clean\s+` + regexp.QuoteMeta(gitFixtureOriginURL))
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

	section := gitReposSummarySection(resp.DryRunCombined)
	if section == "" {
		t.Fatalf("missing GIT REPOS summary section; got:\n%s", resp.DryRunCombined)
	}

	header := metaSectionHeaderLines(section, 1)
	assert.Output(t, header, `---
version: 2
---
GIT REPOS(.backup/git-repo-worktrees.json):
`)
	if !gitReposCapturedAtRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo/worktree count subheader; section:\n%s", section)
	}

	assertGitReposTableHeaders(t, section)
	assertGitReposSummaryContains(t, resp.DryRunCombined,
		".wrk-test/main",
		"repo",
		"main",
		"clean",
		gitFixtureCommitMsg,
		gitFixtureOriginURL,
	)
	if !gitReposOriginRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo row with origin URL; section:\n%s", section)
	}

	members := tarXZListMembers(t, resp.BackupPath)
	if !memberListContains(members, ".backup/git-repo-worktrees.json") {
		t.Fatalf("archive missing .backup/git-repo-worktrees.json; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/git-repo-worktrees.json")
	snap := parseGitRepoWorktreesJSON(t, raw)
	assertGitRepoSnapshotBasics(t, snap, ".wrk-test/main")
	assertGitRepoSnapshotOriginURL(t, snap, ".wrk-test/main", gitFixtureOriginURL)

	for _, repo := range snap.Repos {
		if repo.Path != ".wrk-test/main" {
			continue
		}
		if repo.Branch != "main" {
			t.Fatalf("repo branch = %q, want main", repo.Branch)
		}
		if !strings.Contains(repo.CommitMsg, gitFixtureCommitMsg) {
			t.Fatalf("repo commit_msg = %q, want containing %q", repo.CommitMsg, gitFixtureCommitMsg)
		}
		if repo.Status != "clean" {
			t.Fatalf("repo status = %q, want clean", repo.Status)
		}
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```