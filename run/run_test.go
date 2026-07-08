package run

import (
	"os"
	"testing"

	"golang.org/x/term"
)

func TestAutoKeepAliveWhenNonTTYStdin(t *testing.T) {
	if len(os.Args) > 0 {
		// Harness may pass flags; this test only cares about stdin shape.
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; auto keep-alive applies only to nohup/non-tty launches")
	}
	if !shouldAutoKeepAlive(nil) {
		t.Fatal("empty args with non-tty stdin should auto-delegate to keep-alive")
	}
	if shouldAutoKeepAlive([]string{"keep-alive"}) {
		t.Fatal("explicit subcommand must not auto-delegate")
	}
}