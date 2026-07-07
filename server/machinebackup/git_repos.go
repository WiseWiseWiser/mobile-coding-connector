package machinebackup

import (
	"context"
	"encoding/json"
	"fmt"
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

// ScanGitRepos discovers git repositories under included dot-dir roots.
// When opts.SkipGitDirsScan is true, returns (nil, false, nil).
func ScanGitRepos(home string, dirStats []DirStat, rules ExclusionRules, opts GitScanOptions) (*GitRepoWorktreesSnapshot, bool, error) {
	if opts.SkipGitDirsScan {
		return nil, true, nil
	}
	roots := gitScanRoots(home, dirStats)
	if len(roots) == 0 {
		return emptyGitReposSnapshot(), false, nil
	}

	ctx := context.Background()
	var scanned []scan_repo.Repo
	var rootErrors []scan_repo.RootError
	for _, root := range roots {
		rootRel, err := relPathFromHome(home, root)
		if err != nil {
			return nil, false, fmt.Errorf("rel path for %s: %w", root, err)
		}
		ignoreDirs := gitIgnoreDirsForRoot(home, rootRel, rules)
		result, err := scan_repo.Scan(ctx, scan_repo.Options{
			Roots:         []string{root},
			MaxDepth:      opts.GitDirsScanMaxDepth,
			IgnoreDirs:    ignoreDirs,
			ListWorktrees: true,
		})
		if err != nil {
			return nil, false, fmt.Errorf("scan git repos under %s: %w", rootRel, err)
		}
		scanned = append(scanned, result.Repos...)
		rootErrors = append(rootErrors, result.RootErrors...)
	}

	rel := func(abs string) string {
		return mustRelPathFromHome(home, abs)
	}
	snap := reposnapshot.Build(scan_repo.Result{
		Repos:      scanned,
		RootErrors: rootErrors,
	}, rel)

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

func gitScanRoots(home string, dirStats []DirStat) []string {
	roots := make([]string, 0, len(dirStats))
	for _, st := range dirStats {
		roots = append(roots, filepath.Join(home, filepath.FromSlash(st.Path)))
	}
	sort.Strings(roots)
	return roots
}

func gitIgnoreDirsForRoot(home, rootRel string, rules ExclusionRules) []string {
	rootRel = normalizeRelPath(rootRel)
	var dirs []string
	for _, e := range rules.ExcludedList {
		p := normalizeRelPath(e.Path)
		if p == "" || specialExclusionRules[p] {
			continue
		}
		if p == rootRel || strings.HasPrefix(p, rootRel+"/") {
			dirs = append(dirs, filepath.Join(home, filepath.FromSlash(p)))
		}
	}
	sort.Strings(dirs)
	return dirs
}

func relPathFromHome(home, absPath string) (string, error) {
	rel, err := filepath.Rel(home, absPath)
	if err != nil {
		return "", err
	}
	return normalizeRelPath(rel), nil
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