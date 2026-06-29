package projectpull

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/xgo/support/cmd"
)

func TestBuildPlan_RejectsOversizedUntracked(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "README.md", "dirty\n")
	writeFile(t, dir, "big.bin", strings.Repeat("x", 2*PerFileCapBytes))

	_, err := BuildPlan(PullLocalRequest{Dir: dir})
	if err == nil {
		t.Fatal("expected oversized error")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "1 mb") && !strings.Contains(msg, "1048576") {
		t.Fatalf("expected 1 MB hint: %v", err)
	}
	if !strings.Contains(msg, "include-file") {
		t.Fatalf("expected --include-file hint: %v", err)
	}
}

func TestBuildPlan_IncludeFileAllowsLarge(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "README.md", "dirty\n")
	writeFile(t, dir, "big.bin", strings.Repeat("x", 2*PerFileCapBytes))

	plan, err := BuildPlan(PullLocalRequest{
		Dir:          dir,
		IncludeFiles: []string{"big.bin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.EstimatedBytes <= 0 {
		t.Fatalf("expected positive estimated bytes, got %d", plan.EstimatedBytes)
	}
}

func TestBuildPlan_IncludeFileNotDirty(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "README.md", "dirty\n")

	_, err := BuildPlan(PullLocalRequest{
		Dir:          dir,
		IncludeFiles: []string{"not-in-pull.bin"},
	})
	if err == nil {
		t.Fatal("expected not-part-of-pull error")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "not part") && !strings.Contains(msg, "not-in-pull.bin") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDirtySubmodulePaths(t *testing.T) {
	out := "Entering 'submod'\n M README.md\n"
	dirty := parseDirtySubmodulePaths(out)
	if len(dirty) != 1 || dirty[0] != "submod" {
		t.Fatalf("dirty=%v", dirty)
	}
}

func TestWritePackage_GitApplyRoundTrip(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "README.md", "dirty remote\n")
	writeFile(t, dir, "pulled-untracked.txt", "new\n")

	var buf bytes.Buffer
	if err := WritePackage(&buf, PullLocalRequest{Dir: dir}); err != nil {
		t.Fatal(err)
	}

	commit, err := cmd.Dir(dir).Output("git", "rev-parse", "HEAD^{commit}")
	if err != nil {
		t.Fatal(err)
	}
	commit = strings.TrimSpace(commit)
	wtDir, err := os.MkdirTemp(filepath.Dir(dir), "wt-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(wtDir) })
	if err := cmd.Dir(dir).Run("git", "worktree", "add", "--detach", wtDir, commit); err != nil {
		t.Fatal(err)
	}

	patchBytes, err := gitDiffBytes(dir, commit)
	if err != nil {
		t.Fatal(err)
	}
	patch := string(patchBytes)
	tmpPath := filepath.Join(os.TempDir(), "projectpull-test.patch")
	if err := os.WriteFile(tmpPath, []byte(patch), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpPath) })
	applyCmd := exec.Command("git", "apply", "--whitespace=nowarn", tmpPath)
	applyCmd.Dir = wtDir
	if out, err := applyCmd.CombinedOutput(); err != nil {
		t.Fatalf("git apply failed: %v\n%s\npatch:\n%s", err, out, patch)
	}
}

func TestWritePackage_Layout(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "README.md", "dirty\n")
	writeFile(t, dir, "extra.txt", "new file\n")

	var buf bytes.Buffer
	req := PullLocalRequest{Dir: dir}
	if err := WritePackage(&buf, req); err != nil {
		t.Fatal(err)
	}

	manifest, names, err := ReadPackageManifest(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Commit == "" {
		t.Fatal("empty commit in manifest")
	}
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	for _, want := range []string{"manifest.json", "patch.diff", "untracked/extra.txt"} {
		if !nameSet[want] {
			t.Fatalf("missing tar member %q; have %v", want, names)
		}
	}
	patch, err := PatchDiffFromPackage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(patch, "README.md") {
		t.Fatalf("patch missing readme change: %q", patch)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "projectpull-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "t@example.com")
	gitRun(t, dir, "config", "user.name", "T")
	gitRun(t, dir, "branch", "-M", "main")
	writeFile(t, dir, "README.md", "clean\n")
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "init")
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := cmd.Dir(dir).Run("git", args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}