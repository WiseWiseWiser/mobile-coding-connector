package machinebackup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
			Error:     node.Error,
		}
		for _, wt := range node.Worktrees {
			entry.Worktrees = append(entry.Worktrees, GitWorktreeEntry{
				Path:      wt.Path,
				Branch:    wt.Checkout.Branch,
				CommitSHA: wt.Checkout.CommitSHA,
				CommitMsg: wt.Checkout.CommitMsg,
				Status:    wt.Checkout.Status,
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
		return []string{"  GIT REPOS: (skipped)"}
	}
	if gitRepos == nil || len(gitRepos.Repos) == 0 {
		return []string{"  GIT REPOS: (none)"}
	}
	lines := []string{"  GIT REPOS:"}
	for _, repo := range gitRepos.Repos {
		lines = append(lines, fmt.Sprintf("    %s", repo.Path))
		appendGitCheckoutSummaryLines(&lines, repo.Branch, repo.CommitSHA, repo.Status, repo.CommitMsg, repo.Error, "      ")
		for _, wt := range repo.Worktrees {
			lines = append(lines, fmt.Sprintf("      worktree %s", wt.Path))
			appendGitCheckoutSummaryLines(&lines, wt.Branch, wt.CommitSHA, wt.Status, wt.CommitMsg, wt.Error, "        ")
		}
	}
	return lines
}

func appendGitCheckoutSummaryLines(lines *[]string, branch, sha, status, commitMsg, errMsg, indent string) {
	if errMsg != "" && branch == "" && sha == "" && status == "" {
		*lines = append(*lines, fmt.Sprintf("%serror: %s", indent, errMsg))
		return
	}
	if branch != "" || sha != "" || status != "" {
		*lines = append(*lines, fmt.Sprintf("%sbranch %s  %s  %s", indent, branch, sha, status))
	}
	if commitMsg != "" {
		*lines = append(*lines, fmt.Sprintf("%s%s", indent, commitMsg))
	}
	if errMsg != "" {
		*lines = append(*lines, fmt.Sprintf("%serror: %s", indent, errMsg))
	}
}

func marshalGitReposSnapshot(snap *GitRepoWorktreesSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, nil
	}
	return json.MarshalIndent(snap, "", "  ")
}