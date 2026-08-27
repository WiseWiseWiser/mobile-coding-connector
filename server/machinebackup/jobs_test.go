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

func waitBackupJob(t *testing.T, base, id string, timeout time.Duration) BackupJobView {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last BackupJobView
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/remote-agent/machine/backup/jobs/" + id)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get job status %d: %s", resp.StatusCode, body)
		}
		if err := json.Unmarshal(body, &last); err != nil {
			t.Fatal(err)
		}
		switch last.Status {
		case JobDone, JobError, JobCanceled:
			return last
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for job %s last=%+v", id, last)
	return last
}

func TestBackupJobHTTP_StartPollDownload(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export FAKE=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	reqBody, _ := json.Marshal(BackupStreamRequest{Archive: true, Exclude: []string{}, Include: []string{}})
	resp, err := http.Post(srv.URL+"/api/remote-agent/machine/backup/jobs", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("start status %d: %s", resp.StatusCode, b)
	}
	var started BackupJobView
	if err := json.NewDecoder(resp.Body).Decode(&started); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()
	if started.ID == "" {
		t.Fatal("missing job_id")
	}

	done := waitBackupJob(t, srv.URL, started.ID, 30*time.Second)
	if done.Status != JobDone {
		t.Fatalf("status=%s error=%s", done.Status, done.Error)
	}
	if done.ArchiveToken == "" {
		t.Fatal("missing archive_token")
	}

	dl, err := http.Get(srv.URL + "/api/remote-agent/machine/backup/archive?token=" + done.ArchiveToken)
	if err != nil {
		t.Fatal(err)
	}
	defer dl.Body.Close()
	if dl.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(dl.Body)
		t.Fatalf("download %d: %s", dl.StatusCode, b)
	}
	raw, err := io.ReadAll(dl.Body)
	if err != nil {
		t.Fatal(err)
	}
	wantMagic := []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}
	if len(raw) < 6 || !bytes.Equal(raw[:6], wantMagic) {
		t.Fatalf("archive missing xz magic: %x", raw[:min(6, len(raw))])
	}

	again, err := http.Get(srv.URL + "/api/remote-agent/machine/backup/archive?token=" + done.ArchiveToken)
	if err != nil {
		t.Fatal(err)
	}
	again.Body.Close()
	if again.StatusCode != http.StatusNotFound {
		t.Fatalf("second download want 404 got %d", again.StatusCode)
	}
}

func TestBackupJobHTTP_SingleFlight(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644)

	mux := http.NewServeMux()
	jm := newJobManager(func() string { return home })
	startedCh := make(chan struct{})
	release := make(chan struct{})
	jm.beforeWalk = func(*backupJob) {
		close(startedCh)
		<-release
	}
	registerBackupJobRoutes(mux, jm)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	reqBody, _ := json.Marshal(BackupStreamRequest{Archive: true})
	resp1, err := http.Post(srv.URL+"/api/remote-agent/machine/backup/jobs", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("first start %d", resp1.StatusCode)
	}
	var first BackupJobView
	json.NewDecoder(resp1.Body).Decode(&first)
	resp1.Body.Close()

	<-startedCh
	resp2, err := http.Post(srv.URL+"/api/remote-agent/machine/backup/jobs", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		t.Fatalf("second start want 409 got %d: %s", resp2.StatusCode, b)
	}
	var conflict map[string]any
	json.NewDecoder(resp2.Body).Decode(&conflict)
	resp2.Body.Close()
	if conflict["job_id"] != first.ID {
		t.Fatalf("conflict job_id=%v want %s", conflict["job_id"], first.ID)
	}
	close(release)
}

func TestBackupJob_Cancel(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644)

	jm := newJobManager(func() string { return home })
	startedCh := make(chan struct{})
	jm.beforeWalk = func(job *backupJob) {
		close(startedCh)
		<-job.ctx.Done()
	}
	view, started := jm.start(BackupStreamRequest{Archive: true})
	if !started {
		t.Fatal("expected started")
	}
	<-startedCh
	_, ok, err := jm.cancel(view.ID)
	if !ok || err != nil {
		t.Fatalf("cancel ok=%v err=%v", ok, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := jm.get(view.ID)
		if got.Status == JobCanceled {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := jm.get(view.ID)
	t.Fatalf("want canceled got %+v", got)
}

func TestBackupJob_StaleWatchdog(t *testing.T) {
	stubInstalledToolsSnapshot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0644)

	jm := newJobManager(func() string { return home })
	jm.staleAfter = 40 * time.Millisecond
	jm.heartbeatEvery = 10 * time.Millisecond
	jm.beforeWalk = func(*backupJob) {
		time.Sleep(120 * time.Millisecond)
	}
	view, started := jm.start(BackupStreamRequest{Archive: true})
	if !started {
		t.Fatal("expected started")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := jm.get(view.ID)
		if got.Status == JobError && got.Error != "" {
			if got.Error != "backup job stale: no progress" {
				t.Fatalf("error=%q", got.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := jm.get(view.ID)
	t.Fatalf("want stale error got %+v", got)
}
