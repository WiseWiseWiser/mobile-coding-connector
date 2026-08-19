package testhooks

import (
	"io"
	"strings"
	"sync"

	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

// In-process terminals inject (local-agent terminals list|focus).
// CLI should call TerminalsCapture / TerminalsLayout / TerminalsITermRunning /
// TerminalsFocus when non-nil instead of talking to the daemon. Tests set hooks
// under agentcliMu; ResetInProcessOverrides clears them.
//
// TUI keys: SetTerminalsInput / SetTerminalsKeys script stdin for --tty without
// hijacking os.Stdin. CLI reads TerminalsInput when set.

var (
	terminalsMu sync.Mutex

	terminalsCapture      func() (*kooliterm.Snapshot, error)
	terminalsLayout       func() (*kooliterm.Snapshot, error)
	terminalsITermRunning func() bool
	terminalsFocus        func(ref shelliterm.SessionRef) error
	terminalsInput        io.Reader

	terminalsCaptureCalls int
	terminalsLayoutCalls  int
	terminalsFocusCalled  bool
	terminalsFocusSession string
)

// SetTerminalsCapture installs the deep Capture override for terminals tests.
func SetTerminalsCapture(fn func() (*kooliterm.Snapshot, error)) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsCapture = fn
	terminalsCaptureCalls = 0
}

// SetTerminalsLayout installs the IDs-only Layout probe override.
func SetTerminalsLayout(fn func() (*kooliterm.Snapshot, error)) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsLayout = fn
	terminalsLayoutCalls = 0
}

// SetTerminalsITermRunning installs the iTerm running probe override.
func SetTerminalsITermRunning(fn func() bool) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsITermRunning = fn
}

// SetTerminalsFocus installs the Focus override (not HTTP POST).
func SetTerminalsFocus(fn func(ref shelliterm.SessionRef) error) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsFocus = fn
	terminalsFocusCalled = false
	terminalsFocusSession = ""
}

// TerminalsCapture calls the test Capture hook when set. Returns (nil, false) if unset.
func TerminalsCapture() (*kooliterm.Snapshot, error, bool) {
	terminalsMu.Lock()
	fn := terminalsCapture
	terminalsMu.Unlock()
	if fn == nil {
		return nil, nil, false
	}
	terminalsMu.Lock()
	terminalsCaptureCalls++
	terminalsMu.Unlock()
	snap, err := fn()
	return snap, err, true
}

// TerminalsLayout calls the test Layout hook when set. Returns (nil, false) if unset.
func TerminalsLayout() (*kooliterm.Snapshot, error, bool) {
	terminalsMu.Lock()
	fn := terminalsLayout
	terminalsMu.Unlock()
	if fn == nil {
		return nil, nil, false
	}
	terminalsMu.Lock()
	terminalsLayoutCalls++
	terminalsMu.Unlock()
	snap, err := fn()
	return snap, err, true
}

// TerminalsITermRunning calls the test probe when set. Returns (value, true) if set.
func TerminalsITermRunning() (bool, bool) {
	terminalsMu.Lock()
	fn := terminalsITermRunning
	terminalsMu.Unlock()
	if fn == nil {
		return false, false
	}
	return fn(), true
}

// TerminalsFocus calls the test Focus hook when set. Returns (err, true) if set.
func TerminalsFocus(ref shelliterm.SessionRef) (error, bool) {
	terminalsMu.Lock()
	fn := terminalsFocus
	terminalsMu.Unlock()
	if fn == nil {
		return nil, false
	}
	terminalsMu.Lock()
	terminalsFocusCalled = true
	terminalsFocusSession = ref.SessionID
	terminalsMu.Unlock()
	return fn(ref), true
}

// TerminalsCaptureCalls returns deep Capture invocations since last Set/Reset.
func TerminalsCaptureCalls() int {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	return terminalsCaptureCalls
}

// TerminalsLayoutCalls returns Layout probe invocations since last Set/Reset.
func TerminalsLayoutCalls() int {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	return terminalsLayoutCalls
}

// TerminalsFocusSession returns whether Focus was called and the last session id.
func TerminalsFocusSession() (called bool, sessionID string) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	return terminalsFocusCalled, terminalsFocusSession
}

// SetTerminalsInput installs a scripted key/byte reader for TUI mode (--tty).
// Does not touch os.Stdin. Nil clears.
func SetTerminalsInput(r io.Reader) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsInput = r
}

// SetTerminalsKeys is a convenience for SetTerminalsInput(strings.NewReader(keys)).
// Use named sequences the CLI maps to keys (e.g. "q", "j", "\t", "\r").
func SetTerminalsKeys(keys string) {
	SetTerminalsInput(strings.NewReader(keys))
}

// TerminalsInput returns the scripted TUI reader when set.
func TerminalsInput() (io.Reader, bool) {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	if terminalsInput == nil {
		return nil, false
	}
	return terminalsInput, true
}

func resetTerminalsHooks() {
	terminalsMu.Lock()
	defer terminalsMu.Unlock()
	terminalsCapture = nil
	terminalsLayout = nil
	terminalsITermRunning = nil
	terminalsFocus = nil
	terminalsInput = nil
	terminalsCaptureCalls = 0
	terminalsLayoutCalls = 0
	terminalsFocusCalled = false
	terminalsFocusSession = ""
}
