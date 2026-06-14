package main

import (
	"fmt"
	"strings"
	"testing"
)

type recordRunner struct {
	calls   [][]string
	failAt  int
}

func (r *recordRunner) Run(args ...string) error {
	cp := make([]string, len(args))
	copy(cp, args)
	r.calls = append(r.calls, cp)
	if r.failAt > 0 && len(r.calls) >= r.failAt {
		return fmt.Errorf("mock failure at step %d", len(r.calls))
	}
	return nil
}

func assertCalls(t *testing.T, calls [][]string, wantCount int, step int, wantFirstArg string) {
	t.Helper()
	if len(calls) != wantCount {
		t.Fatalf("got %d calls, want %d", len(calls), wantCount)
	}
	if len(calls) >= step {
		got := calls[step-1][0]
		if got != wantFirstArg {
			t.Fatalf("step %d first arg = %q, want %q", step, got, wantFirstArg)
		}
	}
}

func TestDeploy_Success(t *testing.T) {
	r := &recordRunner{}
	err := runDeploy(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.calls) != 4 {
		t.Fatalf("got %d calls, want 4", len(r.calls))
	}

	// Step 1: auth status
	assertCalls(t, r.calls, 4, 1, "go")
	if !strings.HasSuffix(strings.Join(r.calls[0], " "), "auth status") {
		t.Fatalf("step 1 = %v, want auth status", r.calls[0])
	}

	// Step 2: build
	assertCalls(t, r.calls, 4, 2, "go")
	if !strings.Contains(strings.Join(r.calls[1], " "), "script/bundle") {
		t.Fatalf("step 2 = %v, want bundle build", r.calls[1])
	}

	// Step 3: upload
	assertCalls(t, r.calls, 4, 3, "go")
	args3 := strings.Join(r.calls[2], " ")
	if !strings.Contains(args3, "upload-next") || !strings.Contains(args3, binaryName) {
		t.Fatalf("step 3 = %v, want upload-next %s", r.calls[2], binaryName)
	}

	// Step 4: restart
	assertCalls(t, r.calls, 4, 4, "go")
	if !strings.HasSuffix(strings.Join(r.calls[3], " "), "restart") {
		t.Fatalf("step 4 = %v, want restart", r.calls[3])
	}
}

func TestDeploy_FailAtAuth(t *testing.T) {
	r := &recordRunner{failAt: 1}
	err := runDeploy(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Check auth") {
		t.Fatalf("error = %v, want Check auth", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("got %d calls, want 1 (stopped at auth)", len(r.calls))
	}
}

func TestDeploy_FailAtBuild(t *testing.T) {
	r := &recordRunner{failAt: 2}
	err := runDeploy(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Build binary") {
		t.Fatalf("error = %v, want Build binary", err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("got %d calls, want 2 (stopped at build)", len(r.calls))
	}
}

func TestDeploy_FailAtUpload(t *testing.T) {
	r := &recordRunner{failAt: 3}
	err := runDeploy(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Upload binary") {
		t.Fatalf("error = %v, want Upload binary", err)
	}
	if len(r.calls) != 3 {
		t.Fatalf("got %d calls, want 3 (stopped at upload)", len(r.calls))
	}
}

func TestDryRun(t *testing.T) {
	r := &dryRunRunner{}
	err := runDeploy(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
