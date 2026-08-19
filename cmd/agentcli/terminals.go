package agentcli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/macosapp/itermswitcher"
	"github.com/xhd2015/ai-critic/server/localiterm2"
	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	"github.com/xhd2015/dot-pkgs/go-pkgs/terminal/color"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
	"github.com/xhd2015/less-gen/flags"
	"golang.org/x/term"
)

const terminalsHelp = `Usage: local-agent native-terminals <subcommand> [args...]

List and focus native terminal-emulator windows/tabs (not the PTY 'terminal'
command). iTerm2 on macOS today; other apps and OSes later.

Aliases: native-terminal, native-terms, native-term

Subcommands:
  list [--json] [--refresh] [--tty|--no-tty] [--color|--no-color]
      List live sessions. Uses the local last-good cache and a layout-diff
      refresh. --json prints the last inventory JSON.
      --refresh forces a full recapture of live sessions.
      --tty forces the inline split UI; --no-tty forces batch text.

  focus <session-id>
      Focus a pane by session id.

  -h, --help
      Show this help message.
`

const terminalsListHelp = `Usage: local-agent native-terminals list [options]

List native terminal-emulator sessions (iTerm2 on macOS today).

Default: warm incremental refresh from the local last-good cache.
On a TTY, paints an inline split UI; otherwise batch text.

Options:
  --json       Print the last inventory JSON (no ANSI; always batch)
  --refresh    Force a full recapture of live sessions
  --tty        Force the inline split UI (even if stdout is not a TTY)
  --no-tty     Force batch text output
  --color      Force ANSI color when painting the UI
  --no-color   Disable ANSI color
  -h, --help   Show this help
`

const terminalsFocusHelp = `Usage: local-agent native-terminals focus <session-id>

Focus a live native terminal session by session-id.

Arguments:
  session-id   terminal session UUID

  -h, --help   Show this help
`

func runTerminals(args []string) error {
	if len(args) == 0 {
		fmt.Print(terminalsHelp)
		return nil
	}
	switch args[0] {
	case "list":
		return runTerminalsList(args[1:])
	case "focus":
		return runTerminalsFocus(args[1:])
	case "-h", "--help":
		fmt.Print(terminalsHelp)
		return nil
	default:
		return fmt.Errorf("unknown native-terminals subcommand: %s", args[0])
	}
}

func runTerminalsList(args []string) error {
	var asJSON bool
	var refresh bool
	var forceTTY bool
	var forceNoTTY bool
	var colorFlag bool
	var noColorFlag bool
	args, err := flags.
		Bool("--json", &asJSON).
		Bool("--refresh", &refresh).
		Bool("--tty", &forceTTY).
		Bool("--no-tty", &forceNoTTY).
		Bool("--color", &colorFlag).
		Bool("--no-color", &noColorFlag).
		Help("-h,--help", terminalsListHelp).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("native-terminals list does not accept positional args: %v", args)
	}
	if forceTTY && forceNoTTY {
		return fmt.Errorf("--tty and --no-tty cannot be specified together")
	}
	if _, err := color.ModeFromFlags(colorFlag, noColorFlag); err != nil {
		return err
	}

	h, err := newTerminalsHandler()
	if err != nil {
		return err
	}

	useTUI := false
	if !asJSON {
		if forceTTY {
			useTUI = true
		} else if forceNoTTY {
			useTUI = false
		} else {
			useTUI = term.IsTerminal(int(os.Stdout.Fd()))
		}
	}

	if useTUI {
		return runTerminalsListTUI(h, os.Stdout, refresh)
	}

	inv, err := h.RefreshInventory(refresh)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(inv)
	}
	printTerminalsHuman(&inv)
	return nil
}

// runTerminalsListTUI paints last-good immediately, then probes in the
// background so keys work during AppleScript. Arrow CSI is decoded; stdin
// is raw on a real TTY so arrows do not echo. No alt-screen.
func runTerminalsListTUI(h *localiterm2.Handler, out io.Writer, forceFull bool) error {
	if out == nil {
		out = os.Stdout
	}

	in, hooked := testhooks.TerminalsInput()
	var tty *os.File
	if !hooked {
		in = os.Stdin
		if term.IsTerminal(int(os.Stdin.Fd())) {
			tty = os.Stdin
			old, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err == nil {
				defer func() { _ = term.Restore(int(os.Stdin.Fd()), old) }()
			} else {
				tty = nil
			}
		}
	}

	var lastLines int
	paint := func(state itermswitcher.UIState) error {
		frame := itermswitcher.PaintView(state)
		if tty != nil {
			// raw mode disables ONLCR — CR+LF so the box does not staircase.
			frame = strings.ReplaceAll(frame, "\n", "\r\n")
		}
		if lastLines > 0 {
			if _, err := fmt.Fprintf(out, "\x1b[%dA\x1b[J", lastLines); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, frame); err != nil {
			return err
		}
		lastLines = strings.Count(frame, "\n")
		return nil
	}

	var state itermswitcher.UIState
	if inv, ok := h.CachedInventory(); ok {
		state = itermswitcher.NewUIState(inv, itermswitcher.StatusCached)
	} else {
		state = itermswitcher.NewUIState(h.SeedInventory(), itermswitcher.StatusProbing)
	}
	if err := paint(state); err != nil {
		return err
	}

	type refreshRes struct {
		inv localiterm2.Inventory
		err error
	}
	done := make(chan refreshRes, 1)
	var mu sync.Mutex
	go func() {
		inv, err := h.RefreshInventoryEmit(forceFull, func(partial localiterm2.Inventory) {
			mu.Lock()
			defer mu.Unlock()
			state = itermswitcher.WithInventory(state, partial, itermswitcher.StatusProbing)
			_ = paint(state)
		})
		done <- refreshRes{inv: inv, err: err}
	}()

	keys := make(chan string, 8)
	keyErr := make(chan error, 1)
	go func() {
		br := bufio.NewReader(in)
		for {
			k, err := readTerminalsKey(br, tty)
			if err != nil {
				keyErr <- err
				return
			}
			if k != "" {
				keys <- k
			}
		}
	}()

	apply := func(key string) (doneLoop bool, err error) {
		mu.Lock()
		defer mu.Unlock()
		var act itermswitcher.UIAction
		state, act = itermswitcher.ApplyKey(state, key)
		switch act.Name {
		case "quit":
			return true, nil
		case "focus":
			if act.SessionID == "" {
				return true, fmt.Errorf("session not found")
			}
			return true, h.FocusSession(act.SessionID)
		}
		return false, paint(state)
	}

	// Testhooks stay valid until refresh finishes (live process exit kills the probe).
	finish := func(err error) error {
		if hooked && done != nil {
			<-done
		}
		return err
	}

	for {
		select {
		case res := <-done:
			done = nil
			if res.err != nil {
				return finish(res.err)
			}
			mu.Lock()
			status := itermswitcher.StatusUpToDate
			if forceFull || !res.inv.FromCache {
				status = itermswitcher.StatusIncremental
			}
			state = itermswitcher.WithInventory(state, res.inv, status)
			err := paint(state)
			mu.Unlock()
			if err != nil {
				return finish(err)
			}
		case key := <-keys:
			stop, err := apply(key)
			if err != nil || stop {
				return finish(err)
			}
		case err := <-keyErr:
			if err == io.EOF {
				return finish(nil)
			}
			return finish(err)
		}
	}
}

func runTerminalsFocus(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(terminalsFocusHelp)
		return nil
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("session-id is required")
	}
	sessionID := strings.TrimSpace(args[0])

	h, err := newTerminalsHandler()
	if err != nil {
		return err
	}
	if err := h.FocusSession(sessionID); err != nil {
		return err
	}
	fmt.Printf("focused  %s\n", sessionID)
	return nil
}

// newTerminalsHandler builds an in-process localiterm2.Handler under testhooks
// HOME (or real home). Never dials the daemon.
func newTerminalsHandler() (*localiterm2.Handler, error) {
	home, err := testhooks.UserHomeDir()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(home) == "" {
		return nil, fmt.Errorf("home directory is empty")
	}
	aiCritic := filepath.Join(home, ".ai-critic")
	h := &localiterm2.Handler{
		CachePath:   filepath.Join(aiCritic, "iterm-inventory-cache.json"),
		Notes:       localiterm2.NewNoteStore(filepath.Join(aiCritic, "iterm-bookmarks.json")),
		SpaceLabels: localiterm2.NewSpaceLabelStore(filepath.Join(aiCritic, "space-labels.json")),
	}
	h.Capture = func() (*kooliterm.Snapshot, error) {
		if snap, err, ok := testhooks.TerminalsCapture(); ok {
			return snap, err
		}
		var doc *localiterm2.NotesDocument
		if h.Notes != nil {
			doc = h.Notes.Document()
		}
		snap, _, err := kooliterm.CaptureSnapshotWith(kooliterm.CaptureOpts{
			NoEnrich: !localiterm2.NeedsAgentEnrich(doc),
		})
		return snap, err
	}
	h.Layout = func() (*kooliterm.Snapshot, error) {
		if snap, err, ok := testhooks.TerminalsLayout(); ok {
			return snap, err
		}
		return h.DefaultLayout()
	}
	h.ITermRunning = func() bool {
		if v, ok := testhooks.TerminalsITermRunning(); ok {
			return v
		}
		return true
	}
	h.Focus = func(ref shelliterm.SessionRef) error {
		if err, ok := testhooks.TerminalsFocus(ref); ok {
			return err
		}
		return shelliterm.Focus(ref, nil)
	}
	return h, nil
}

func printTerminalsHuman(inv *localiterm2.Inventory) {
	if inv == nil || !inv.ITermRunning {
		fmt.Fprintln(os.Stdout, itermswitcher.FormatEmptyITerm())
		return
	}
	for _, d := range inv.Desktops {
		fmt.Fprintln(os.Stdout, itermswitcher.FormatSidebarDesktopTitle(d.SpaceIndex, d.Label))
		for _, s := range d.Sessions {
			fmt.Fprintln(os.Stdout, "  "+itermswitcher.FormatSessionPrimary(s.SessionName, s.Cwd, s.SessionID))
		}
	}
	if inv.FromCache {
		fmt.Fprintln(os.Stdout, "up to date")
	} else {
		fmt.Fprintln(os.Stdout, "incremental")
	}
}
