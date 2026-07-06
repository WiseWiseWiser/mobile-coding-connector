package machinebackup

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverAndExclusions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	files := map[string]string{
		".bashrc":                   "export FAKE=1\n",
		".cache/junk":               "cache\n",
		".npm/x/package.json":       "{}\n",
		".cargo/config.toml":        "[source]\n",
		".cargo/registry/db/idx":    "registry\n",
		"Projects/visible.txt":      "visible\n",
		".ai-critic/ai-models.json": "{}\n",
	}
	for rel, content := range files {
		full := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(home, ".local", "bin", "tool-link")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../.bashrc", link); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".bashrc", ".ai-critic/ai-models.json", ".cargo/config.toml", ".local/bin/tool-link"} {
		if !contains(plan.Included, want) {
			t.Fatalf("missing included %q: %v", want, plan.Included)
		}
	}
	for _, absent := range []string{".cache/junk", ".npm/x/package.json", ".cargo/registry/db/idx", "Projects/visible.txt"} {
		if contains(plan.Included, absent) {
			t.Fatalf("unexpected included %q", absent)
		}
	}
	for _, want := range []string{".cache", ".npm", ".cargo/registry", ".backup"} {
		if !containsExcluded(plan.Excluded, want) {
			t.Fatalf("missing excluded %q: %v", want, plan.Excluded)
		}
	}
}

func TestCustomExclude(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".docker", "config")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(home, []string{".docker"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(plan.Included, ".docker/config") {
		t.Fatalf("custom exclude failed: %v", plan.Included)
	}
	if !containsExcluded(plan.Excluded, ".docker") {
		t.Fatalf("expected .docker in excluded: %v", plan.Excluded)
	}
}

func stubInstalledToolsSnapshot(t *testing.T) {
	t.Helper()
	oldSnap := buildInstalledToolsSnapshotFn
	buildInstalledToolsSnapshotFn = func() ([]byte, error) {
		return []byte("{\n  \"captured_at\": \"2020-01-01T00:00:00Z\",\n  \"tools\": []\n}\n"), nil
	}
	t.Cleanup(func() { buildInstalledToolsSnapshotFn = oldSnap })
}

func TestTarXZRoundtrip(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	models := filepath.Join(home, ".ai-critic", "ai-models.json")
	if err := os.MkdirAll(filepath.Dir(models), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(models, []byte(`{"models":[]}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteArchive(&buf, home, nil, nil); err != nil {
		t.Fatal(err)
	}
	magic := []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}
	if !bytes.Equal(buf.Bytes()[:6], magic) {
		t.Fatalf("missing xz magic: % x", buf.Bytes()[:6])
	}

	manifest, entries, err := ReadArchive(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Home != home {
		t.Fatalf("manifest home %q want %q", manifest.Home, home)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Header.Name)
	}
	for _, want := range []string{".bashrc", ".ai-critic/ai-models.json"} {
		if !contains(names, want) {
			t.Fatalf("archive missing %q: %v", want, names)
		}
	}
}

func TestRestoreIdenticalSkip(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	models := filepath.Join(home, ".ai-critic", "ai-models.json")
	if err := os.MkdirAll(filepath.Dir(models), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(models, []byte(`{"models":[]}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildRestorePlan(home, bytes.NewReader(archive.Bytes()), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range plan.Entries {
		if entry.Action != "skip" {
			t.Fatalf("expected skip for identical home, got %v", entry)
		}
	}
}

func TestRestoreApplyUpdatesChangedFile(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("mutated\n"), 0644); err != nil {
		t.Fatal(err)
	}

	plan, err := ApplyRestore(home, bytes.NewReader(archive.Bytes()), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundUpdate := false
	for _, entry := range plan.Entries {
		if entry.Path == ".bashrc" && entry.Action == "update" {
			foundUpdate = true
		}
	}
	if !foundUpdate {
		t.Fatalf("expected update for .bashrc: %v", plan.Entries)
	}
	got, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "export FAKE=1\n" {
		t.Fatalf("restore content %q", got)
	}
}

func TestWalkAccumulatesBytes(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("1234567890"), 0644); err != nil {
		t.Fatal(err)
	}
	models := filepath.Join(home, ".ai-critic", "ai-models.json")
	if err := os.MkdirAll(filepath.Dir(models), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(models, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(home, ".ai-critic", "nested", "file.txt")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".profile")
	if err := os.Symlink(".bashrc", link); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var bashrcBytes int64
	for _, f := range plan.DotFiles {
		switch f.Path {
		case ".bashrc":
			if f.Bytes != 10 {
				t.Fatalf(".bashrc bytes=%d want 10", f.Bytes)
			}
			bashrcBytes = f.Bytes
		case ".profile":
			if !f.Symlink || f.Bytes != 0 {
				t.Fatalf(".profile symlink=%v bytes=%d", f.Symlink, f.Bytes)
			}
		}
	}
	if bashrcBytes == 0 {
		t.Fatal("missing .bashrc in dot files")
	}

	var aiCritic *DirStat
	for i := range plan.DirStats {
		if plan.DirStats[i].Path == ".ai-critic" {
			aiCritic = &plan.DirStats[i]
			break
		}
	}
	if aiCritic == nil {
		t.Fatal("missing .ai-critic dir stat")
	}
	if aiCritic.Bytes != 8 {
		t.Fatalf(".ai-critic bytes=%d want 8 (3+5)", aiCritic.Bytes)
	}
	if plan.DotFilesTotal.Bytes != 10 {
		t.Fatalf("dot files total bytes=%d want 10", plan.DotFilesTotal.Bytes)
	}
	if plan.GrandTotal.Bytes != 18 {
		t.Fatalf("grand total bytes=%d want 18", plan.GrandTotal.Bytes)
	}
}

func TestBackupPlanStreamEmitsProgressThenDone(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := newFlushRecorder()
	if err := BackupPlanStream(rec, home, nil, nil); err != nil {
		t.Fatal(err)
	}
	body := rec.body.String()
	if !strings.Contains(body, `"type":"section"`) && !strings.Contains(body, `"type": "section"`) {
		t.Fatalf("missing section frame: %s", body)
	}
	if !strings.Contains(body, `"layer":"dot_file"`) && !strings.Contains(body, `"layer": "dot_file"`) {
		t.Fatalf("missing dot_file progress: %s", body)
	}
	if !strings.Contains(body, `"type":"done"`) && !strings.Contains(body, `"type": "done"`) {
		t.Fatalf("missing done frame: %s", body)
	}
	if !strings.Contains(body, "grand_total") {
		t.Fatalf("done payload missing sizes: %s", body)
	}
}

func TestRestorePlanStreamDryRun(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil); err != nil {
		t.Fatal(err)
	}

	rec := newFlushRecorder()
	if err := RestorePlanStream(rec, home, bytes.NewReader(archive.Bytes()), nil, nil, true); err != nil {
		t.Fatal(err)
	}
	body := rec.body.String()
	if !strings.Contains(body, `"status":"skip"`) && !strings.Contains(body, `"status": "skip"`) {
		t.Fatalf("missing skip progress: %s", body)
	}
	if !strings.Contains(body, "total_entries") {
		t.Fatalf("missing restore summary: %s", body)
	}
}

type flushRecorder struct {
	headers http.Header
	body    bytes.Buffer
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{headers: make(http.Header)}
}

func (f *flushRecorder) Header() http.Header       { return f.headers }
func (f *flushRecorder) Write(p []byte) (int, error) { return f.body.Write(p) }
func (f *flushRecorder) WriteHeader(statusCode int)  {}
func (f *flushRecorder) Flush()                      {}

func TestIncludeReenablesBuiltinExclude(t *testing.T) {
	home := t.TempDir()
	cache := filepath.Join(home, ".cache", "junk")
	if err := os.MkdirAll(filepath.Dir(cache), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("cache\n"), 0644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(home, nil, []string{".cache"})
	if err != nil {
		t.Fatal(err)
	}
	if containsExcluded(plan.Excluded, ".cache") {
		t.Fatalf(".cache should be included via --include: %v", plan.Excluded)
	}
	if !contains(plan.Included, ".cache/junk") {
		t.Fatalf("expected .cache/junk in included: %v", plan.Included)
	}
}

func TestWriteArchiveIncludesBackupMeta(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	metaDir := filepath.Join(home, ".backup")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "config.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteArchive(&buf, home, nil, nil); err != nil {
		t.Fatal(err)
	}
	_, entries, err := ReadArchive(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Header.Name)
	}
	for _, want := range []string{
		".backup/config.json",
		".backup/installed.json",
		".backup/ENV",
		".backup/config.json.machine.bak",
	} {
		if !contains(names, want) {
			t.Fatalf("archive missing %q: %v", want, names)
		}
	}
}

func TestRestoreSkipsMetaRestoresMachineBak(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildRestorePlan(home, bytes.NewReader(archive.Bytes()), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range plan.Entries {
		if strings.HasPrefix(entry.Path, ".backup/") &&
			!strings.HasSuffix(entry.Path, ".machine.bak") &&
			entry.Path != ".backup/config.json" {
			// only .machine.bak targets are restored under .backup
		}
		if entry.Path == ".backup/config.json" || entry.Path == ".backup/installed.json" || entry.Path == ".backup/ENV" {
			t.Fatalf("meta snapshot should not be restored: %v", entry)
		}
	}
}

func TestSymlinkNotFollowed(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("target\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".profile")
	if err := os.Symlink(".bashrc", link); err != nil {
		t.Fatal(err)
	}

	rules := MergeExclusions(nil, nil)
	res, err := discover(home, rules)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, m := range res.Members {
		if m.RelPath == ".profile" && m.IsSymlink && m.Linkname == ".bashrc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected symlink member, got %v", res.Members)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if strings.TrimPrefix(item, "./") == want {
			return true
		}
	}
	return false
}

func containsExcluded(items []ExcludePathEntry, want string) bool {
	for _, item := range items {
		if item.Path == want {
			return true
		}
	}
	return false
}