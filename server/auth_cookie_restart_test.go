package server

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestRestartPreservesConfiguredCookieArgument(t *testing.T) {
	previousArgs, previousExec := os.Args, execProcess
	t.Cleanup(func() { os.Args, execProcess = previousArgs, previousExec })
	want := []string{"ai-critic-server", "--port", "23712", "--auth-cookie-name", "pi-agent-token"}
	os.Args = append([]string(nil), want...)
	intercepted := errors.New("exec intercepted, no process replacement")
	var got []string
	execProcess = func(bin string, argv []string, env []string) error {
		got = append([]string(nil), argv...)
		return intercepted
	}
	plan := &restartPlan{bin: "/fixture/next-server", released: make(chan struct{})}
	close(plan.released)
	if err := performPendingRestart(plan); !errors.Is(err, intercepted) {
		t.Fatalf("restart did not use the intercepted exec: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("self-upgrade lost startup arguments: %v", got)
	}
}
