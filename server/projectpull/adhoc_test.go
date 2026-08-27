package projectpull

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInspectAllowsCleanRepo(t *testing.T) {
	dir := initTempRepo(t)
	mustGit(t, dir, "commit", "--allow-empty", "-m", "init")
	insp, err := InspectRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !insp.IsClean {
		t.Fatal("expected clean")
	}
	if insp.Commit == "" {
		t.Fatal("expected commit")
	}
	if insp.FullTreeBytes < 0 {
		t.Fatalf("full tree bytes %d", insp.FullTreeBytes)
	}
}

func TestWriteBundleAndDownloadPackage(t *testing.T) {
	dir := initTempRepo(t)
	writeFileContent(t, filepath.Join(dir, "readme.txt"), "hello\n")
	mustGit(t, dir, "add", "readme.txt")
	mustGit(t, dir, "commit", "-m", "add readme")

	var bundle bytes.Buffer
	if err := WriteBundle(&bundle, dir); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	if bundle.Len() == 0 {
		t.Fatal("empty bundle")
	}

	var dl bytes.Buffer
	if err := WriteDownloadPackage(&dl, dir); err != nil {
		t.Fatalf("WriteDownloadPackage: %v", err)
	}
	if dl.Len() == 0 {
		t.Fatal("empty download package")
	}
}

func TestInspectDirtyEstimates(t *testing.T) {
	dir := initTempRepo(t)
	writeFileContent(t, filepath.Join(dir, "a.txt"), "a\n")
	mustGit(t, dir, "add", "a.txt")
	mustGit(t, dir, "commit", "-m", "a")
	writeFileContent(t, filepath.Join(dir, "a.txt"), "b\n")
	insp, err := InspectRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if insp.IsClean {
		t.Fatal("expected dirty")
	}
	if insp.DirtyTracked < 1 && insp.DirtyUntracked < 1 {
		t.Fatalf("expected dirty counts, got tracked=%d untracked=%d", insp.DirtyTracked, insp.DirtyUntracked)
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "t@example.com")
	mustGit(t, dir, "config", "user.name", "t")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFileContent(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
