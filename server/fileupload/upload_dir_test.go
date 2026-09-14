package fileupload

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestApplyUploadDirArchive_CreatesAndOverrides(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "apps", "myapp")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dest, "a.txt")
	if err := os.WriteFile(existing, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(root, "payload.tar.xz")
	packTestArchive(t, archive, map[string]string{
		"a.txt":     "new\n",
		"sub/b.txt": "bravo\n",
	})

	resp, err := applyUploadDirArchive(UploadDirApplyRequest{
		ArchivePath:   archive,
		Dest:          dest,
		NoOverride:    false,
		DeleteArchive: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if resp.FileCount != 2 {
		t.Fatalf("file_count=%d want 2", resp.FileCount)
	}
	if len(resp.Overridden) != 1 || resp.Overridden[0] != "a.txt" {
		t.Fatalf("overridden=%v want [a.txt]", resp.Overridden)
	}
	assertFile(t, filepath.Join(dest, "a.txt"), "new\n")
	assertFile(t, filepath.Join(dest, "sub", "b.txt"), "bravo\n")
	if _, err := os.Stat(archive); !os.IsNotExist(err) {
		t.Fatalf("archive should be deleted, err=%v", err)
	}
}

func TestApplyUploadDirArchive_NoOverrideConflict(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "a.txt"), []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "payload.tar.xz")
	packTestArchive(t, archive, map[string]string{"a.txt": "new\n"})

	_, err := applyUploadDirArchive(UploadDirApplyRequest{
		ArchivePath: archive,
		Dest:        dest,
		NoOverride:  true,
	})
	if err == nil {
		t.Fatal("expected no-override error")
	}
	assertFile(t, filepath.Join(dest, "a.txt"), "old\n")
}

func TestSanitizeTarRelPathRejectsDotDot(t *testing.T) {
	if _, err := sanitizeTarRelPath("../x"); err == nil {
		t.Fatal("expected error")
	}
}

func packTestArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	xzw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xzw)
	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xzw.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s: got %q want %q", path, got, want)
	}
}
