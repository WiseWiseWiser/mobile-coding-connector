package machinebackup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xhd2015/dot-pkgs/go-pkgs/git/reposnapshot"
	"github.com/xhd2015/dot-pkgs/go-pkgs/git/scan_repo"
)

const (
	metaGitReposName = "git-repo-worktrees.json"
	gitReposVersion  = "1.0"
)

// ScanGitRepos discovers git repositories under HOME.
// When opts.SkipGitDirsScan is true, returns (nil, false, nil).
func ScanGitRepos(home string, opts GitScanOptions) (*GitRepoWorktreesSnapshot, bool, error) {
	if opts.SkipGitDirsScan {
		return nil, true, nil
	}

	ctx := context.Background()
	result, err := scan_repo.Scan(ctx, scan_repo.Options{
		Roots:         []string{home},
		MaxDepth:      opts.GitDirsScanMaxDepth,
		ListWorktrees: true,
	})
	if err != nil {
		return nil, false, fmt.Errorf("scan git repos under HOME: %w", err)
	}

	rel := func(abs string) string {
		return mustRelPathFromHome(home, abs)
	}
	snap := reposnapshot.Build(result, rel)

	return gitReposSnapshotFromBuild(snap), false, nil
}

func emptyGitReposSnapshot() *GitRepoWorktreesSnapshot {
	return &GitRepoWorktreesSnapshot{
		Version:    gitReposVersion,
		CapturedAt: time.Now().UTC(),
		Repos:      []GitRepoEntry{},
	}
}

func gitReposSnapshotFromBuild(snap reposnapshot.Snapshot) *GitRepoWorktreesSnapshot {
	repos := make([]GitRepoEntry, 0, len(snap.Nodes))
	for _, node := range snap.Nodes {
		entry := GitRepoEntry{
			Path:      node.Path,
			Branch:    node.Checkout.Branch,
			CommitSHA: node.Checkout.CommitSHA,
			CommitMsg: node.Checkout.CommitMsg,
			Status:    node.Checkout.Status,
			OriginURL: node.Checkout.OriginURL,
			Error:     node.Error,
		}
		for _, wt := range node.Worktrees {
			entry.Worktrees = append(entry.Worktrees, GitWorktreeEntry{
				Path:      wt.Path,
				Branch:    wt.Checkout.Branch,
				CommitSHA: wt.Checkout.CommitSHA,
				CommitMsg: wt.Checkout.CommitMsg,
				Status:    wt.Checkout.Status,
				OriginURL: wt.Checkout.OriginURL,
				Error:     wt.Error,
			})
		}
		repos = append(repos, entry)
	}
	return &GitRepoWorktreesSnapshot{
		Version:    gitReposVersion,
		CapturedAt: time.Now().UTC(),
		Repos:      repos,
	}
}

func relPathFromHome(home, absPath string) (string, error) {
	rel, err := filepath.Rel(home, absPath)
	if err != nil {
		return "", err
	}
	rel = normalizeRelPath(rel)
	return foldRelPathFirstSegmentCase(home, rel), nil
}

// foldRelPathFirstSegmentCase lowercases the first path segment when it refers to
// the same path on a case-insensitive volume (e.g. Projects/demo vs projects/demo).
func foldRelPathFirstSegmentCase(home, rel string) string {
	if rel == "" || strings.HasPrefix(rel, ".") {
		return rel
	}
	parts := strings.Split(rel, "/")
	if len(parts) == 0 || parts[0] == "" {
		return rel
	}
	first := parts[0]
	if first == strings.ToLower(first) {
		return rel
	}
	lowerFirst := strings.ToLower(first[:1]) + first[1:]
	orig := filepath.Join(home, filepath.FromSlash(rel))
	alt := filepath.Join(home, filepath.FromSlash(lowerFirst))
	if len(parts) > 1 {
		alt = filepath.Join(alt, filepath.FromSlash(strings.Join(parts[1:], "/")))
	}
	if pathsSameFile(orig, alt) {
		parts[0] = lowerFirst
		return strings.Join(parts, "/")
	}
	return rel
}

func pathsSameFile(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(ia, ib)
}

func mustRelPathFromHome(home, absPath string) string {
	rel, err := relPathFromHome(home, absPath)
	if err != nil {
		return normalizeRelPath(absPath)
	}
	return rel
}

func formatGitReposSummaryLines(gitRepos *GitRepoWorktreesSnapshot, skipped bool) []string {
	if skipped {
		return []string{gitReposDryRunHeader + " (skipped)"}
	}
	if gitRepos == nil || len(gitRepos.Repos) == 0 {
		return []string{gitReposDryRunHeader + " (none)"}
	}

	repos := append([]GitRepoEntry(nil), gitRepos.Repos...)
	sort.Slice(repos, func(i, j int) bool { return repos[i].Path < repos[j].Path })

	worktreeCount := 0
	for _, repo := range repos {
		worktreeCount += len(repo.Worktrees)
	}

	lines := []string{
		gitReposDryRunHeader,
		fmt.Sprintf("    captured_at: %s  (%d repo, %d worktree)",
			formatMetaCapturedAt(gitRepos.CapturedAt), len(repos), worktreeCount),
		gitReposTableColumnHeader,
	}
	for _, repo := range repos {
		lines = append(lines, formatGitReposCheckoutRow(
			"repo", repo.Path, repo.Branch, repo.CommitSHA, repo.Status, repo.OriginURL, repo.CommitMsg, repo.Error,
		))
		for _, wt := range repo.Worktrees {
			lines = append(lines, formatGitReposCheckoutRow(
				"worktree", wt.Path, wt.Branch, wt.CommitSHA, wt.Status, wt.OriginURL, wt.CommitMsg, wt.Error,
			))
		}
	}
	return appendMetaSectionTerminator(lines)
}

func formatGitReposCheckoutRow(kind, path, branch, sha, status, originURL, commitMsg, errMsg string) string {
	const rowIndent = "    "
	if isGitCheckoutErrorOnly(branch, sha, status, errMsg) {
		return fmt.Sprintf("%s%-10s%-24s%-14s%-10s%s",
			rowIndent, kind, path, "", "", "error: "+errMsg)
	}
	origin := "(none)"
	if originURL != "" {
		origin = originURL
	}
	return fmt.Sprintf("%s%-10s%-24s%-14s%-10s%-20s%-40s%s",
		rowIndent, kind, path, branch, shortGitCommitSHA(sha), status, origin, commitMsg)
}

func isGitCheckoutErrorOnly(branch, sha, status, errMsg string) bool {
	return errMsg != "" && branch == "" && sha == "" && status == ""
}

func shortGitCommitSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func marshalGitReposSnapshot(snap *GitRepoWorktreesSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, nil
	}
	return json.MarshalIndent(snap, "", "  ")
}