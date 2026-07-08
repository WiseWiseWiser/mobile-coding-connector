package machinebackup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestorePlanStreamApplyEmitsClassifyingAndApplying(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	const original = "export FAKE=1\n"
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil, GitScanOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("mutated after backup\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := newFlushRecorder()
	if err := RestorePlanStream(rec, home, bytes.NewReader(archive.Bytes()), nil, nil, false); err != nil {
		t.Fatal(err)
	}
	body := rec.body.String()
	if !strings.Contains(body, `"message":"CLASSIFYING"`) && !strings.Contains(body, `"message": "CLASSIFYING"`) {
		t.Fatalf("missing CLASSIFYING section: %s", body)
	}
	if !strings.Contains(body, `"message":"APPLYING"`) && !strings.Contains(body, `"message": "APPLYING"`) {
		t.Fatalf("missing APPLYING section: %s", body)
	}
	if !strings.Contains(body, `"status":"update"`) && !strings.Contains(body, `"status": "update"`) {
		t.Fatalf("missing update progress: %s", body)
	}

	got, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf(".bashrc not restored from archive: got %q want %q", got, original)
	}
}

func TestRestorePlanStreamDryRunOmitsApplying(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil, GitScanOptions{}); err != nil {
		t.Fatal(err)
	}

	rec := newFlushRecorder()
	if err := RestorePlanStream(rec, home, bytes.NewReader(archive.Bytes()), nil, nil, true); err != nil {
		t.Fatal(err)
	}
	body := rec.body.String()
	if !strings.Contains(body, `"message":"CLASSIFYING"`) && !strings.Contains(body, `"message": "CLASSIFYING"`) {
		t.Fatalf("missing CLASSIFYING section: %s", body)
	}
	if strings.Contains(body, `"message":"APPLYING"`) || strings.Contains(body, `"message": "APPLYING"`) {
		t.Fatalf("dry-run must not emit APPLYING section: %s", body)
	}
}