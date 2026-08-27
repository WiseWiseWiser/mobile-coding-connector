package machinebackup

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestoreJobHTTP_Apply(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err := WriteArchive(&archive, home, nil, nil, GitScanOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("mutated\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/remote-agent/machine/restore/jobs", "application/x-xz", bytes.NewReader(archive.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("start %d: %s", resp.StatusCode, b)
	}
	var started RestoreJobView
	if err := json.NewDecoder(resp.Body).Decode(&started); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()

	deadline := time.Now().Add(30 * time.Second)
	var last RestoreJobView
	for time.Now().Before(deadline) {
		gr, err := http.Get(srv.URL + "/api/remote-agent/machine/restore/jobs/" + started.ID)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(gr.Body)
		gr.Body.Close()
		if err := json.Unmarshal(body, &last); err != nil {
			t.Fatal(err)
		}
		if last.Status == JobDone || last.Status == JobError || last.Status == JobCanceled {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if last.Status != JobDone {
		t.Fatalf("status=%s error=%s", last.Status, last.Error)
	}
	got, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "export FAKE=1\n" {
		t.Fatalf(".bashrc=%q", got)
	}
}
