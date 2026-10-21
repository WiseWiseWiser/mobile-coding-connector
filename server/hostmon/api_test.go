package hostmon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setDefaultForTest(t *testing.T, m *Manager) {
	t.Helper()
	defaultMu.Lock()
	prev := def
	def = m
	defaultMu.Unlock()
	t.Cleanup(func() {
		_ = m.Stop()
		defaultMu.Lock()
		def = prev
		defaultMu.Unlock()
	})
}

func TestAPIStatusAndSamples(t *testing.T) {
	m := New(t.TempDir())
	m.collect = func(prev *counters) (Sample, *counters, error) {
		return Sample{T: time.Now().Format(time.RFC3339), Load1: 1}, &counters{}, nil
	}
	setDefaultForTest(t, m)
	if err := appendSample(m.dir, Sample{T: "2026-10-21T06:40:00Z", Load1: 6.2, Top: []ProcSample{{PID: 7, Comm: "node"}}}); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterAPI(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/remote-agent/hostmon/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Running || status.AutoStart || status.Latest == nil || status.Latest.Top[0].Comm != "node" {
		t.Fatalf("status = %+v", status)
	}

	resp2, err := http.Get(srv.URL + "/api/remote-agent/hostmon/samples?lines=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var body struct {
		Samples []Sample `json:"samples"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Samples) != 1 || body.Samples[0].Top[0].PID != 7 {
		t.Fatalf("samples = %+v", body.Samples)
	}
}
