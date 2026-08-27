package projectpull

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitops "github.com/xhd2015/gitops/git"
	"github.com/xhd2015/xgo/support/cmd"
)

// InspectRequest is the JSON body for POST .../pull-local/inspect.
type InspectRequest struct {
	Dir string `json:"dir"`
}

// InspectResult describes a remote git worktree for adhoc pull-local planning.
type InspectResult struct {
	Dir              string `json:"dir"`
	Commit           string `json:"commit"`
	Branch           string `json:"branch"`
	OriginURL        string `json:"origin_url"`
	IsClean          bool   `json:"is_clean"`
	AheadOfOrigin    int    `json:"ahead_of_origin"`
	HasOrigin        bool   `json:"has_origin"`
	DirtyTracked     int    `json:"dirty_tracked"`
	DirtyUntracked   int    `json:"dirty_untracked"`
	FullTreeBytes    int64  `json:"full_tree_bytes"`
	DirtyPackageEst  int64  `json:"dirty_package_est_bytes"`
	BundleNeededHint bool   `json:"bundle_needed_hint"`
}

// InspectRepo returns remote git metadata for pull-local adhoc modes.
// Unlike BuildPlan, a clean worktree is allowed.
func InspectRepo(dir string) (*InspectResult, error) {
	if err := validateDir(dir); err != nil {
		return nil, err
	}
	inspect, err := gitops.InspectWorktree(dir)
	if err != nil {
		return nil, err
	}
	if inspect == nil || !inspect.IsRepo {
		return nil, fmt.Errorf("dir is not a git repository: %s", dir)
	}

	commit, err := cmd.Dir(dir).Output("git", "rev-parse", "HEAD^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve commit: %w", err)
	}
	commit = strings.TrimSpace(commit)

	originURL := ""
	hasOrigin := false
	if originOut, err := cmd.Dir(dir).Output("git", "remote", "get-url", "origin"); err == nil {
		originURL = strings.TrimSpace(originOut)
		hasOrigin = originURL != ""
	}

	branch := strings.TrimSpace(inspect.Branch)
	if branch == "" {
		branch = "detached"
	}

	ahead := 0
	if hasOrigin {
		ahead, err = countAheadOfOrigin(dir, branch)
		if err != nil {
			// Non-fatal: tip may be unrelated to origin; treat as needing bundle.
			ahead = -1
		}
	}

	fullBytes, err := estimateFullTreeBytes(dir)
	if err != nil {
		return nil, err
	}

	var dirtyEst int64
	if !inspect.IsClean {
		st, err := collectPullState(dir, nil, DefaultMaxSizeBytes*16)
		if err == nil {
			dirtyEst = st.estimatedBytes
		}
	}

	bundleHint := !hasOrigin || ahead > 0 || ahead < 0

	return &InspectResult{
		Dir:              dir,
		Commit:           commit,
		Branch:           branch,
		OriginURL:        originURL,
		IsClean:          inspect.IsClean,
		AheadOfOrigin:    ahead,
		HasOrigin:        hasOrigin,
		DirtyTracked:     inspect.Changed + inspect.Renamed + inspect.Deleted,
		DirtyUntracked:   inspect.Added,
		FullTreeBytes:    fullBytes,
		DirtyPackageEst:  dirtyEst,
		BundleNeededHint: bundleHint,
	}, nil
}

func countAheadOfOrigin(dir, branch string) (int, error) {
	candidates := []string{"origin/" + branch, "origin/main", "origin/master"}
	var base string
	for _, ref := range candidates {
		if _, err := cmd.Dir(dir).Output("git", "rev-parse", "--verify", ref); err == nil {
			base = ref
			break
		}
	}
	if base == "" {
		return -1, fmt.Errorf("no origin base ref")
	}
	out, err := cmd.Dir(dir).Output("git", "rev-list", "--count", "HEAD", "--not", base)
	if err != nil {
		return -1, err
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &n); err != nil {
		return -1, err
	}
	return n, nil
}

func estimateFullTreeBytes(dir string) (int64, error) {
	out, err := cmd.Dir(dir).Output("git", "ls-files", "-z", "-c", "-o", "--exclude-standard")
	if err != nil {
		return 0, fmt.Errorf("list worktree files: %w", err)
	}
	var total int64
	for _, rel := range strings.Split(out, "\x00") {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		info, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}
	return total, nil
}
