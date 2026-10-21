package hostmon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendSurvivesReopenAndRotates(t *testing.T) {
	dir := t.TempDir()
	old := MaxLogBytes
	MaxLogBytes = 180
	t.Cleanup(func() { MaxLogBytes = old })

	first := Sample{T: time.Date(2026, 10, 21, 6, 40, 0, 0, time.UTC).Format(time.RFC3339), Load1: 1}
	if err := appendSample(dir, first); err != nil {
		t.Fatal(err)
	}
	got, err := readSamples(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Load1 != 1 {
		t.Fatalf("samples = %+v", got)
	}

	for i := 0; i < 20; i++ {
		if err := appendSample(dir, Sample{T: first.T, Load1: float64(i + 2)}); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(filepath.Join(dir, logName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > MaxLogBytes+400 {
		t.Fatalf("current log size = %d, cap %d", info.Size(), MaxLogBytes)
	}
	if _, err := os.Stat(filepath.Join(dir, logName+".1")); err != nil {
		t.Fatalf("rotated file: %v", err)
	}
	all, err := readSamples(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 2 {
		t.Fatalf("kept %d samples", len(all))
	}
	last, err := readSamples(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(last) != 1 || last[0].Load1 != all[len(all)-1].Load1 {
		t.Fatalf("tail = %+v, want last of %+v", last, all[len(all)-1])
	}
}

func TestManagerStartStopPersistsWhileStopped(t *testing.T) {
	m := New(t.TempDir())
	m.interval = 15 * time.Millisecond
	m.collect = func(prev *counters) (Sample, *counters, error) {
		sample := Sample{T: "2026-10-21T06:40:00Z", Load1: 6.2, Top: []ProcSample{{PID: 1842, Comm: "cloudflared"}}}
		return sample, &counters{}, nil
	}
	if m.AutoStart() || m.Running() {
		t.Fatal("new manager must be stopped and not auto-start")
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		samples, err := m.Samples(1)
		if err != nil {
			t.Fatal(err)
		}
		if len(samples) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	if m.Running() {
		t.Fatal("still running after Stop")
	}
	samples, err := m.Samples(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 || samples[0].Top[0].PID != 1842 || samples[0].Top[0].Comm != "cloudflared" {
		t.Fatalf("samples = %+v", samples)
	}
	if err := m.SetAutoStart(true); err != nil {
		t.Fatal(err)
	}
	if !m.AutoStart() || m.Running() {
		t.Fatalf("auto=%v running=%v, want auto on and still stopped", m.AutoStart(), m.Running())
	}
}

func TestBootStartsOnlyWhenEnabled(t *testing.T) {
	m := New(t.TempDir())
	m.collect = func(prev *counters) (Sample, *counters, error) {
		return Sample{}, &counters{}, nil
	}
	m.Boot()
	if m.Running() {
		t.Fatal("Boot started a disabled monitor")
	}
	if err := m.SetAutoStart(true); err != nil {
		t.Fatal(err)
	}
	m.Boot()
	if !m.Running() {
		t.Fatal("Boot did not start an enabled monitor")
	}
	_ = m.Stop()
}
