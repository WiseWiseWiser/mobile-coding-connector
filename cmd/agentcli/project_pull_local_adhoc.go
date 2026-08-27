package agentcli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/xgo/support/cmd"
)

const pullLocalFullTreeWarnBytes = 100 << 20 // 100 MiB

func looksLikeRemoteAbsPath(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~/")
}

func resolveAdhocOrRegistered(cli *client.Client, target string, forceAdhoc bool) (project *client.ProjectInfo, adhoc bool, err error) {
	if forceAdhoc {
		return syntheticAdhocProject(cli, target)
	}
	p, err := resolveProjectTarget(cli, target)
	if err == nil {
		return p, false, nil
	}
	// Fall back to adhoc when target looks like an absolute remote path.
	if looksLikeRemoteAbsPath(target) {
		return syntheticAdhocProject(cli, target)
	}
	return nil, false, err
}

func syntheticAdhocProject(cli *client.Client, dir string) (*client.ProjectInfo, bool, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, true, fmt.Errorf("adhoc remote dir is required")
	}
	insp, err := cli.PullLocalInspect(dir)
	if err != nil {
		return nil, true, err
	}
	name := filepath.Base(strings.TrimSuffix(insp.Dir, "/"))
	if name == "" || name == "." || name == "/" {
		name = "adhoc"
	}
	return &client.ProjectInfo{
		ID:      "adhoc:" + insp.Dir,
		Name:    name,
		Dir:     insp.Dir,
		RepoURL: insp.OriginURL,
		GitStatus: client.ProjectGitStatus{
			IsClean: insp.IsClean,
			Branch:  insp.Branch,
			Commit:  insp.Commit,
			Added:   insp.DirtyUntracked,
			Changed: insp.DirtyTracked,
		},
	}, true, nil
}

func ensureLocalGitRepo(cli *client.Client, localPath string, insp *client.PullLocalInspect) error {
	gitDir := filepath.Join(localPath, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		return nil
	}
	if insp.OriginURL == "" {
		return fmt.Errorf("local path %s is not a git repo and remote has no origin to clone", localPath)
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(localPath); err == nil {
		// Path exists but is not a git repo.
		entries, _ := os.ReadDir(localPath)
		if len(entries) > 0 {
			return fmt.Errorf("local path %s exists and is not an empty git clone destination", localPath)
		}
	}
	fmt.Printf("Cloning %s into %s\n", insp.OriginURL, localPath)
	if err := cmd.New().Run("git", "clone", insp.OriginURL, localPath); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

func ensureLocalTip(cli *client.Client, localPath, remoteDir, tip string, insp *client.PullLocalInspect) error {
	if tip == "" {
		return fmt.Errorf("remote tip commit is empty")
	}
	if err := cmd.Dir(localPath).Run("git", "fetch", "--all", "--prune"); err != nil {
		// Non-fatal when origin missing remotely; may still use bundle.
		fmt.Fprintf(os.Stderr, "warning: git fetch in local repo: %v\n", err)
	}
	if localHasCommit(localPath, tip) {
		return nil
	}
	if !insp.BundleNeededHint && insp.HasOrigin {
		// Retry fetch origin specifically.
		_ = cmd.Dir(localPath).Run("git", "fetch", "origin")
		if localHasCommit(localPath, tip) {
			return nil
		}
	}
	fmt.Println("Fetching missing commits via remote git bundle…")
	body, err := cli.PullLocalBundle(remoteDir)
	if err != nil {
		return fmt.Errorf("download bundle: %w", err)
	}
	defer body.Close()
	tmp, err := os.CreateTemp("", "pull-local-bundle-*.bundle")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := cmd.Dir(localPath).Run("git", "fetch", tmpPath, tip); err != nil {
		// Broader fetch of all refs in bundle.
		if err2 := cmd.Dir(localPath).Run("git", "fetch", tmpPath); err2 != nil {
			return fmt.Errorf("git fetch bundle: %v (retry: %v)", err, err2)
		}
	}
	if !localHasCommit(localPath, tip) {
		return fmt.Errorf("tip commit %s still missing after bundle fetch", tip)
	}
	return nil
}

func localHasCommit(localPath, tip string) bool {
	err := cmd.Dir(localPath).Run("git", "cat-file", "-e", tip+"^{commit}")
	return err == nil
}

func applyDirtyPackageIfNeeded(cli *client.Client, project *client.ProjectInfo, worktreePath string, includeFiles []string, maxSizeBytes int64, insp *client.PullLocalInspect) error {
	if insp.IsClean {
		return nil
	}
	pullReq := client.PullLocalRequest{
		Dir:          project.Dir,
		IncludeFiles: includeFiles,
		MaxSizeBytes: maxSizeBytes,
	}
	pkgBody, err := cli.PullLocalPackage(pullReq)
	if err != nil {
		return err
	}
	defer pkgBody.Close()
	return applyPullLocalPackage(pkgBody, worktreePath)
}

func runAdhocGitFetch(cli *client.Client, project *client.ProjectInfo, localPath string, includeFiles []string, maxSizeBytes int64, noTruncate, dryRun bool) error {
	insp, err := cli.PullLocalInspect(project.Dir)
	if err != nil {
		return err
	}
	branch := insp.Branch
	if branch == "" || branch == "detached" {
		branch = "detached"
	}
	worktreePath, err := allocateWorktreeDir(cli.Server, project.Name, project.Dir, branch)
	if err != nil {
		return err
	}

	if dryRun {
		printAdhocGitFetchDryRun(project, localPath, worktreePath, insp, noTruncate)
		return nil
	}

	if err := ensureLocalGitRepo(cli, localPath, insp); err != nil {
		return err
	}
	localOrigin, err := localGitOriginURL(localPath)
	if err == nil && insp.OriginURL != "" && !sameGitOrigin(localOrigin, insp.OriginURL) {
		return fmt.Errorf("git origin mismatch: local %q vs remote %q", localOrigin, insp.OriginURL)
	}
	if err := ensureLocalTip(cli, localPath, project.Dir, insp.Commit, insp); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return err
	}
	branchName := worktreeBranchName(worktreePath)
	if err := cmd.Dir(localPath).Run("git", "worktree", "add", "-b", branchName, worktreePath, insp.Commit); err != nil {
		return fmt.Errorf("create local worktree: %w", err)
	}
	if err := applyDirtyPackageIfNeeded(cli, project, worktreePath, includeFiles, maxSizeBytes, insp); err != nil {
		return err
	}
	if !noTruncate {
		if err := cli.PullLocalTruncate(project.Dir, insp.Commit); err != nil {
			return err
		}
	}
	fmt.Printf("Pulled (git-fetch) from %s into worktree %s\n", project.Dir, worktreePath)
	if noTruncate {
		fmt.Println("Remote repository was not truncated (--no-truncate-remote).")
	} else {
		fmt.Println("Remote repository was reset to a clean state.")
	}
	return nil
}

func printAdhocGitFetchDryRun(project *client.ProjectInfo, localPath, worktreePath string, insp *client.PullLocalInspect, noTruncate bool) {
	fmt.Printf("dry-run: adhoc pull-local  mode=git-fetch\n\n")
	fmt.Printf("  remote dir:   %s\n", project.Dir)
	fmt.Printf("  origin:       %s\n", insp.OriginURL)
	fmt.Printf("  tip:          %s\n", insp.Commit)
	fmt.Printf("  branch:       %s\n", insp.Branch)
	fmt.Printf("  ahead origin: %d\n", insp.AheadOfOrigin)
	fmt.Printf("  dirty:        tracked=%d untracked=%d clean=%v\n", insp.DirtyTracked, insp.DirtyUntracked, insp.IsClean)
	fmt.Printf("  bundle hint:  %v\n", insp.BundleNeededHint)
	fmt.Printf("  local repo:   %s\n", localPath)
	fmt.Printf("  worktree:     %s\n", worktreePath)
	fmt.Printf("  dirty pkg:    %d bytes (est)\n", insp.DirtyPackageEst)
	fmt.Printf("  full-tree:    %d bytes (est; not used in git-fetch)\n", insp.FullTreeBytes)
	if noTruncate {
		fmt.Printf("  remote:       would keep dirty state (--no-truncate-remote)\n")
	} else {
		fmt.Printf("  remote:       would reset --hard and clean -fd\n")
	}
}

func runAdhocDownload(cli *client.Client, project *client.ProjectInfo, localPath string, noTruncate, dryRun bool) error {
	if strings.TrimSpace(localPath) == "" {
		return fmt.Errorf("--local-path is required for --mode download")
	}
	insp, err := cli.PullLocalInspect(project.Dir)
	if err != nil {
		return err
	}
	if dryRun {
		printAdhocDownloadDryRun(project, localPath, insp, noTruncate)
		return nil
	}
	if insp.FullTreeBytes > pullLocalFullTreeWarnBytes {
		fmt.Fprintf(os.Stderr, "warning: full-tree estimate %d bytes exceeds 100M\n", insp.FullTreeBytes)
	}
	body, err := cli.PullLocalDownload(project.Dir)
	if err != nil {
		return err
	}
	defer body.Close()
	if err := applyDownloadPackage(body, localPath); err != nil {
		return err
	}
	if !noTruncate {
		if err := cli.PullLocalTruncate(project.Dir, insp.Commit); err != nil {
			return err
		}
	}
	fmt.Printf("Downloaded worktree from %s into %s\n", project.Dir, localPath)
	if noTruncate {
		fmt.Println("Remote repository was not truncated (--no-truncate-remote).")
	} else {
		fmt.Println("Remote repository was reset to a clean state.")
	}
	return nil
}

func printAdhocDownloadDryRun(project *client.ProjectInfo, localPath string, insp *client.PullLocalInspect, noTruncate bool) {
	fmt.Printf("dry-run: adhoc pull-local  mode=download\n\n")
	fmt.Printf("  remote dir:   %s\n", project.Dir)
	fmt.Printf("  origin:       %s\n", insp.OriginURL)
	fmt.Printf("  tip:          %s\n", insp.Commit)
	fmt.Printf("  branch:       %s\n", insp.Branch)
	fmt.Printf("  full-tree:    %d bytes (est)\n", insp.FullTreeBytes)
	if insp.FullTreeBytes > pullLocalFullTreeWarnBytes {
		fmt.Printf("  warning:      estimate exceeds 100M\n")
	}
	fmt.Printf("  local path:   %s\n", localPath)
	if noTruncate {
		fmt.Printf("  remote:       would keep dirty state (--no-truncate-remote)\n")
	} else {
		fmt.Printf("  remote:       would reset --hard and clean -fd\n")
	}
}

func applyDownloadPackage(r io.Reader, localPath string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("read download package: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	tmp, err := os.MkdirTemp("", "pull-local-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	var hasBundle bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := hdr.Name
		switch {
		case name == "repo.bundle":
			dest := filepath.Join(tmp, "repo.bundle")
			if err := writeReaderFile(dest, tr, 0644); err != nil {
				return err
			}
			hasBundle = true
		case strings.HasPrefix(name, "files/"):
			rel := strings.TrimPrefix(name, "files/")
			dest := filepath.Join(tmp, "files", filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			if err := writeReaderFile(dest, tr, 0644); err != nil {
				return err
			}
		default:
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return err
			}
		}
	}
	if !hasBundle {
		return fmt.Errorf("download package missing repo.bundle")
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(localPath); err == nil {
		entries, _ := os.ReadDir(localPath)
		if len(entries) > 0 {
			return fmt.Errorf("local path %s is not empty", localPath)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	bundlePath := filepath.Join(tmp, "repo.bundle")
	if err := cmd.New().Run("git", "clone", bundlePath, localPath); err != nil {
		return fmt.Errorf("git clone bundle: %w", err)
	}
	// Overlay worktree files (captures dirty/untracked state at tip).
	filesRoot := filepath.Join(tmp, "files")
	if st, err := os.Stat(filesRoot); err == nil && st.IsDir() {
		if err := copyDirOverlay(filesRoot, localPath); err != nil {
			return err
		}
	}
	return nil
}

func writeReaderFile(dest string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, r)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyDirOverlay(srcRoot, dstRoot string) error {
	return filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(dstRoot, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		return writeReaderFile(dest, in, info.Mode().Perm())
	})
}

