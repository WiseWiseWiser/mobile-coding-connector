package synccmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StatusOpts configures Status with injectable serve probe (parallel-safe).
type StatusOpts struct {
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string
	Name         string // empty → all pairs
	ServeOK      func() error
	// Check, when true, runs Unison for each pair (may transfer) then reloads state.
	Check bool
	// Run hooks used when Check is true (nil → production defaults).
	Exec          ExecFunc
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	RemotePathOK  func(remote string) error
	SkipDoctor    bool // when Check: skip doctor if true
	Stdout        io.Writer
	Stderr        io.Writer
	Context       context.Context
}

// StatusItem is one pair's identity, serve flag, and sync outcome metadata.
type StatusItem struct {
	Name     string
	Local    string
	Remote   string
	ServeOK  bool
	LastRun  string // lastRunAt or "never" (compat with existing doctests)
	LastExit *int
	// Display fields for CLI table.
	Sync     string // never | synced | ok | failed
	LastSync string // timestamp or "—"
	Detail   string
}

// StatusReport is the list of status items.
type StatusReport struct {
	Items []StatusItem
}

// pairState is the on-disk last-run metadata under {StoreDir}/state/<name>.json.
type pairState struct {
	LastRunAt        string `json:"lastRunAt"`
	ExitCode         *int   `json:"exitCode"`
	Message          string `json:"message"`
	Outcome          string `json:"outcome,omitempty"`
	ItemsTransferred int    `json:"itemsTransferred,omitempty"`
	ItemsFailed      int    `json:"itemsFailed,omitempty"`
	ItemsSkipped     int    `json:"itemsSkipped,omitempty"`
	DurationMs       int64  `json:"durationMs,omitempty"`
}

// Status loads pairs and optional state files; name empty lists all pairs.
// When Check is set, runs Unison per pair first (serve required for real checks).
func Status(opts StatusOpts) (StatusReport, error) {
	if opts.Check {
		if err := runStatusCheck(opts); err != nil {
			// Still try to report whatever state we have, but surface check error.
			rep, _ := statusFromStore(opts)
			return rep, err
		}
	}
	return statusFromStore(opts)
}

func runStatusCheck(opts StatusOpts) error {
	store := &Store{Dir: opts.StoreDir}
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	var names []string
	if opts.Name != "" {
		names = []string{opts.Name}
	} else {
		for _, p := range cfg.Pairs {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	var firstErr error
	for _, name := range names {
		_, rerr := RunPair(RunOpts{
			StoreDir:      opts.StoreDir,
			UnisonDir:     opts.UnisonDir,
			SSHConfigDir:  opts.SSHConfigDir,
			Name:          name,
			SkipDoctor:    opts.SkipDoctor,
			Exec:          opts.Exec,
			Stdout:        opts.Stdout,
			Stderr:        opts.Stderr,
			Context:       opts.Context,
			LocalVersion:  opts.LocalVersion,
			RemoteVersion: opts.RemoteVersion,
			ServeOK:       opts.ServeOK,
			RemotePathOK:  opts.RemotePathOK,
		})
		if rerr != nil && firstErr == nil {
			firstErr = fmt.Errorf("check %s: %w", name, rerr)
		}
	}
	return firstErr
}

func statusFromStore(opts StatusOpts) (StatusReport, error) {
	store := &Store{Dir: opts.StoreDir}
	cfg, err := store.Load()
	if err != nil {
		return StatusReport{}, err
	}

	var pairs []Pair
	if opts.Name != "" {
		p, err := GetPair(store, opts.Name)
		if err != nil {
			return StatusReport{}, err
		}
		pairs = []Pair{*p}
	} else {
		pairs = append([]Pair(nil), cfg.Pairs...)
	}

	serveOK := false
	if opts.ServeOK != nil {
		serveOK = opts.ServeOK() == nil
	} else {
		serveOK = defaultServeOK(opts.SSHConfigDir) == nil
	}

	rep := StatusReport{Items: make([]StatusItem, 0, len(pairs))}
	for _, p := range pairs {
		item := StatusItem{
			Name:     p.Name,
			Local:    p.Local,
			Remote:   p.Remote,
			ServeOK:  serveOK,
			LastRun:  "never",
			Sync:     OutcomeNever,
			LastSync: "—",
			Detail:   "no successful run yet",
		}
		st, serr := loadPairState(opts.StoreDir, p.Name)
		if serr == nil && st != nil {
			if st.LastRunAt != "" {
				item.LastRun = st.LastRunAt
			}
			if st.ExitCode != nil {
				code := *st.ExitCode
				item.LastExit = &code
			}
			item.Sync, item.LastSync, item.Detail = outcomeFromState(st)
			// Display LAST SYNC in local time (state file stays UTC).
			item.LastSync = formatLocalTime(item.LastSync)
			if item.LastRun != "never" {
				item.LastRun = formatLocalTime(item.LastRun)
			}
		}
		rep.Items = append(rep.Items, item)
	}
	return rep, nil
}

// formatLocalTime converts RFC3339 / RFC3339Nano (often UTC Z) to local wall time.
// Non-timestamps (e.g. "—", "never") are returned unchanged.
func formatLocalTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || strings.EqualFold(s, "never") {
		return s
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return s
	}
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}

func loadPairState(storeDir, name string) (*pairState, error) {
	path := filepath.Join(storeDir, "state", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st pairState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("state %s: %w", name, err)
	}
	// Compat: older files may have exitCode as number only; already *int via Unmarshal.
	// Also accept bare exitCode without pointer when message-only files exist.
	if st.ExitCode == nil {
		// Try re-parse flexible form
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			if b, ok := raw["exitCode"]; ok {
				var n int
				if json.Unmarshal(b, &n) == nil {
					st.ExitCode = &n
				}
			}
		}
	}
	return &st, nil
}

// writePairStateFull writes enriched last-run state after a run/check.
func writePairStateFull(storeDir, name string, exitCode int, message, outcome string, transferred, failed, skipped int, dur time.Duration) error {
	dir := filepath.Join(storeDir, "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	code := exitCode
	st := pairState{
		LastRunAt:        time.Now().UTC().Format(time.RFC3339),
		ExitCode:         &code,
		Message:          message,
		Outcome:          outcome,
		ItemsTransferred: transferred,
		ItemsFailed:      failed,
		ItemsSkipped:     skipped,
		DurationMs:       dur.Milliseconds(),
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644)
}