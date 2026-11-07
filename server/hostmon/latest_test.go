package hostmon

import (
	"fmt"
	"testing"
)

// Latest must read only the log tail and still decode the final line, even
// when the tail window starts in the middle of an earlier sample.
func TestLatestReadsLogTail(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{dir: dir}

	want := ""
	for i := 0; i < 400; i++ {
		sample := Sample{T: fmt.Sprintf("2026-10-21T06:40:%02dZ", i%60), Load1: float64(i)}
		for j := 0; j < 10; j++ {
			sample.Top = append(sample.Top, ProcSample{PID: j, Comm: fmt.Sprintf("proc-%03d-%04d", i, j)})
		}
		if err := appendSample(dir, sample); err != nil {
			t.Fatal(err)
		}
		want = sample.T
	}
	if size := logBytes(dir); size < 64*1024 {
		t.Fatalf("log too small to exercise the tail window: %d bytes", size)
	}

	got, ok := m.Latest()
	if !ok {
		t.Fatal("Latest() found no sample")
	}
	if got.T != want {
		t.Fatalf("Latest() T = %q, want %q", got.T, want)
	}
	if len(got.Top) != 10 {
		t.Fatalf("Latest() top = %d entries, want 10", len(got.Top))
	}
}

func TestLatestMissingLog(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	if _, ok := m.Latest(); ok {
		t.Fatal("Latest() must report no sample for a fresh directory")
	}
}
