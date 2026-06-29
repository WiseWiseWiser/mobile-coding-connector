package projectpull

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitops "github.com/xhd2015/gitops/git"
	"github.com/xhd2015/xgo/support/cmd"
)

const (
	PerFileCapBytes     = 1 << 20
	DefaultMaxSizeBytes = 64 << 20
)

// PullLocalRequest is the JSON body for POST /api/remote-agent/project/pull-local.
type PullLocalRequest struct {
	Dir          string   `json:"dir"`
	DryRun       bool     `json:"dry_run"`
	IncludeFiles []string `json:"include_files"`
	MaxSizeBytes int64    `json:"max_size_bytes"`
}

// OversizedFileEntry describes one path over the per-file cap.
type OversizedFileEntry struct {
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Included bool   `json:"included"`
}

// PullLocalPlan is returned when dry_run is true.
type PullLocalPlan struct {
	Dir             string               `json:"dir"`
	Commit          string               `json:"commit"`
	Branch          string               `json:"branch"`
	OriginURL       string               `json:"origin_url"`
	IsClean         bool                 `json:"is_clean"`
	TrackedFiles    int                  `json:"tracked_files"`
	UntrackedFiles  int                  `json:"untracked_files"`
	DeletedFiles    int                  `json:"deleted_files"`
	SubmodulesOK    bool                 `json:"submodules_ok"`
	DirtySubmodules []string             `json:"dirty_submodules"`
	EstimatedBytes  int64                `json:"estimated_bytes"`
	OversizedFiles  []OversizedFileEntry `json:"oversized_files"`
	WithinMaxSize   bool                 `json:"within_max_size"`
}

// Manifest is written into the pull-local tar.gz package.
type Manifest struct {
	Commit           string   `json:"commit"`
	Branch           string   `json:"branch"`
	OriginURL        string   `json:"origin_url"`
	UntrackedFiles   []string `json:"untracked_files"`
	TotalBytes       int64    `json:"total_bytes"`
	IncludeFilesUsed []string `json:"include_files_used"`
}

// TruncateRequest is the JSON body for POST .../pull-local/truncate.
type TruncateRequest struct {
	Dir    string `json:"dir"`
	Commit string `json:"commit"`
}

type pullState struct {
	dir            string
	commit         string
	branch         string
	originURL      string
	inspect        *gitops.WorktreeInspect
	untracked      []string
	modifiedPaths  []string
	patchDiff      []byte
	estimatedBytes int64
	oversized      []OversizedFileEntry
	maxSize        int64
}

func resolveMaxSize(max int64) int64 {
	if max <= 0 {
		return DefaultMaxSizeBytes
	}
	return max
}

func normalizeRelPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}

func validateDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("dir is required")
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("dir does not exist: %s", dir)
		}
		return fmt.Errorf("stat dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dir is not a directory: %s", dir)
	}
	inside, err := gitops.IsInsideGit(dir)
	if err != nil {
		return err
	}
	if !inside {
		return fmt.Errorf("dir is not a git repository: %s", dir)
	}
	return nil
}

// BuildPlan inspects the repo and applies pull-local guards.
func BuildPlan(req PullLocalRequest) (*PullLocalPlan, error) {
	if err := validateDir(req.Dir); err != nil {
		return nil, err
	}
	st, err := collectPullState(req.Dir, req.IncludeFiles, resolveMaxSize(req.MaxSizeBytes))
	if err != nil {
		return nil, err
	}
	return st.toPlan(), nil
}

// WritePackage streams a gzip tar of manifest.json, patch.diff, and untracked/* members.
func WritePackage(w io.Writer, req PullLocalRequest) error {
	if err := validateDir(req.Dir); err != nil {
		return err
	}
	st, err := collectPullState(req.Dir, req.IncludeFiles, resolveMaxSize(req.MaxSizeBytes))
	if err != nil {
		return err
	}

	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	manifest := Manifest{
		Commit:           st.commit,
		Branch:           st.branch,
		OriginURL:        st.originURL,
		UntrackedFiles:   append([]string(nil), st.untracked...),
		TotalBytes:       st.estimatedBytes,
		IncludeFilesUsed: append([]string(nil), req.IncludeFiles...),
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := writeTarBytes(tw, "manifest.json", manifestJSON, 0644); err != nil {
		return err
	}
	if err := writeTarBytes(tw, "patch.diff", st.patchDiff, 0644); err != nil {
		return err
	}
	for _, rel := range st.untracked {
		abs := filepath.Join(st.dir, filepath.FromSlash(rel))
		tarName := "untracked/" + rel
		if err := writeTarFile(tw, tarName, abs); err != nil {
			return fmt.Errorf("add untracked %s: %w", rel, err)
		}
	}
	return nil
}

// TruncateWorktree runs git reset --hard and git clean -fd in dir.
func TruncateWorktree(dir, commit string) error {
	if err := validateDir(dir); err != nil {
		return err
	}
	ref := strings.TrimSpace(commit)
	if ref == "" {
		ref = "HEAD"
	}
	if err := cmd.Dir(dir).Run("git", "reset", "--hard", ref); err != nil {
		return fmt.Errorf("git reset --hard: %w", err)
	}
	if err := cmd.Dir(dir).Run("git", "clean", "-fd"); err != nil {
		return fmt.Errorf("git clean -fd: %w", err)
	}
	return nil
}

func collectPullState(dir string, includeFiles []string, maxSize int64) (*pullState, error) {
	inspect, err := gitops.InspectWorktree(dir)
	if err != nil {
		return nil, err
	}
	if inspect == nil || !inspect.IsRepo {
		return nil, fmt.Errorf("dir is not a git repository: %s", dir)
	}
	if inspect.IsClean {
		return nil, fmt.Errorf("nothing to pull: remote worktree is clean")
	}
	if err := checkSubmodulesClean(dir); err != nil {
		return nil, err
	}

	commit, err := cmd.Dir(dir).Output("git", "rev-parse", "HEAD^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve commit: %w", err)
	}
	commit = strings.TrimSpace(commit)

	originURL := ""
	if originOut, err := cmd.Dir(dir).Output("git", "remote", "get-url", "origin"); err == nil {
		originURL = strings.TrimSpace(originOut)
	}

	patchDiff, err := gitDiffBytes(dir, commit)
	if err != nil {
		return nil, err
	}

	untrackedOut, err := cmd.Dir(dir).Output("git", "ls-files", "-o", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("list untracked: %w", err)
	}
	var untracked []string
	for _, line := range strings.Split(untrackedOut, "\n") {
		line = normalizeRelPath(line)
		if line != "" {
			untracked = append(untracked, line)
		}
	}

	dirtyPaths, modifiedPaths, err := dirtyPathsFromPorcelain(dir)
	if err != nil {
		return nil, err
	}
	for _, u := range untracked {
		dirtyPaths[u] = struct{}{}
	}

	includeNorm := make(map[string]struct{})
	for _, inc := range includeFiles {
		n := normalizeRelPath(inc)
		if n == "" {
			continue
		}
		includeNorm[n] = struct{}{}
		if _, ok := dirtyPaths[n]; !ok {
			return nil, fmt.Errorf("%s is not part of pull (not in dirty set); check --include-file", n)
		}
	}

	perPathBytes := make(map[string]int64)
	var oversized []OversizedFileEntry
	for path := range dirtyPaths {
		sz, err := dirtyPathBytes(dir, path)
		if err != nil {
			return nil, err
		}
		perPathBytes[path] = sz
		_, included := includeNorm[path]
		if sz > PerFileCapBytes && !included {
			oversized = append(oversized, OversizedFileEntry{Path: path, Bytes: sz, Included: false})
		}
	}
	if len(oversized) > 0 {
		return nil, fmt.Errorf(
			"file exceeds 1 MB limit (%d bytes max per file); add to .gitignore or pass --include-file for: %s",
			PerFileCapBytes,
			oversized[0].Path,
		)
	}

	var estimated int64
	estimated += int64(len(patchDiff))
	for _, rel := range untracked {
		estimated += perPathBytes[rel]
	}
	for _, rel := range modifiedPaths {
		if containsString(untracked, rel) {
			continue
		}
		estimated += perPathBytes[rel]
	}

	if estimated > maxSize {
		return nil, fmt.Errorf(
			"pull package size %d bytes exceeds limit %d bytes (default 64 MB); pass a higher --max-size",
			estimated,
			maxSize,
		)
	}

	branch := strings.TrimSpace(inspect.Branch)
	if branch == "" {
		branch = "detached"
	}

	return &pullState{
		dir:            dir,
		commit:         commit,
		branch:         branch,
		originURL:      originURL,
		inspect:        inspect,
		untracked:      untracked,
		modifiedPaths:  modifiedPaths,
		patchDiff:      patchDiff,
		estimatedBytes: estimated,
		oversized:      oversized,
		maxSize:        maxSize,
	}, nil
}

func (st *pullState) toPlan() *PullLocalPlan {
	tracked := st.inspect.Changed + st.inspect.Renamed
	return &PullLocalPlan{
		Dir:             st.dir,
		Commit:          st.commit,
		Branch:          st.branch,
		OriginURL:       st.originURL,
		IsClean:         false,
		TrackedFiles:    tracked,
		UntrackedFiles:  st.inspect.Added,
		DeletedFiles:    st.inspect.Deleted,
		SubmodulesOK:    true,
		DirtySubmodules: []string{},
		EstimatedBytes:  st.estimatedBytes,
		OversizedFiles:  st.oversized,
		WithinMaxSize:   st.estimatedBytes <= st.maxSize,
	}
}

func dirtyPathsFromPorcelain(dir string) (map[string]struct{}, []string, error) {
	out, err := cmd.Dir(dir).Output("git", "status", "--porcelain=v1")
	if err != nil {
		return nil, nil, err
	}
	dirty := make(map[string]struct{})
	var modified []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		pathPart := strings.TrimSpace(line[3:])
		if pathPart == "" {
			continue
		}
		rel := normalizeRelPath(pathPart)
		if idx := strings.Index(rel, " -> "); idx >= 0 {
			rel = normalizeRelPath(strings.TrimSpace(rel[idx+4:]))
		}
		if rel == "" {
			continue
		}
		dirty[rel] = struct{}{}
		if line[0:2] != "??" {
			modified = append(modified, rel)
		}
	}
	return dirty, modified, nil
}

func dirtyPathBytes(dir, rel string) (int64, error) {
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

func checkSubmodulesClean(dir string) error {
	if !hasTrackedGitmodules(dir) {
		return nil
	}
	out, err := cmd.Dir(dir).Output("git", "submodule", "foreach", "--recursive", "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("remote submodule status check failed: %w", err)
	}
	dirty := parseDirtySubmodulePaths(out)
	if len(dirty) == 0 {
		return nil
	}
	return fmt.Errorf("dirty submodule(s): %s", strings.Join(dirty, ", "))
}

// gitDiffBytes returns git diff output without trimming the trailing newline (xgo cmd.Output strips it).
func gitDiffBytes(dir, commit string) ([]byte, error) {
	c := exec.Command("git", "diff", commit)
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

func hasTrackedGitmodules(dir string) bool {
	c := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", ".gitmodules")
	return c.Run() == nil
}

func parseDirtySubmodulePaths(foreachOutput string) []string {
	var dirty []string
	var current string
	for _, line := range strings.Split(foreachOutput, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Entering ") {
			current = parseSubmoduleEntering(trimmed)
			continue
		}
		if current == "" || trimmed == "" {
			continue
		}
		if len(line) >= 3 && line[2] == ' ' {
			if !containsString(dirty, current) {
				dirty = append(dirty, current)
			}
		}
	}
	return dirty
}

func parseSubmoduleEntering(line string) string {
	start := strings.Index(line, "'")
	end := strings.LastIndex(line, "'")
	if start < 0 || end <= start {
		return ""
	}
	return line[start+1 : end]
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func writeTarBytes(tw *tar.Writer, name string, data []byte, mode int64) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    mode,
		Size:    int64(len(data)),
		ModTime: tarHeaderTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func writeTarFile(tw *tar.Writer, tarName, absPath string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = tarName
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// ReadPackageManifest parses manifest.json from a pull-local tar.gz stream.
func ReadPackageManifest(r io.Reader) (*Manifest, []string, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	var names []string
	var manifest *Manifest
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		names = append(names, hdr.Name)
		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, nil, err
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, nil, err
			}
			manifest = &m
		}
	}
	if manifest == nil {
		return nil, names, fmt.Errorf("manifest.json missing from package")
	}
	return manifest, names, nil
}

// PatchDiffFromPackage reads patch.diff from an in-memory tar.gz (test helper).
func PatchDiffFromPackage(tgz []byte) (string, error) {
	gr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return "", err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("patch.diff not found")
		}
		if err != nil {
			return "", err
		}
		if hdr.Name == "patch.diff" {
			data, err := io.ReadAll(tr)
			return string(data), err
		}
	}
}