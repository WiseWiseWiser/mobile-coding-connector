package synccmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StatusOpts configures Status with injectable serve probe (parallel-safe).
type StatusOpts struct {
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string
	Name         string // empty → all pairs
	ServeOK      func() error
}

// StatusItem is one pair's identity, serve flag, and last-run metadata.
type StatusItem struct {
	Name     string
	Local    string
	Remote   string
	ServeOK  bool
	LastRun  string
	LastExit *int // optional; populated from state file when present
}

// StatusReport is the list of status items.
type StatusReport struct {
	Items []StatusItem
}

// pairState is the on-disk last-run metadata under {StoreDir}/state/<name>.json.
type pairState struct {
	LastRunAt string `json:"lastRunAt"`
	ExitCode  *int   `json:"exitCode"`
	Message   string `json:"message"`
}

// Status loads pairs and optional state files; name empty lists all pairs.
func Status(opts StatusOpts) (StatusReport, error) {
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
			Name:    p.Name,
			Local:   p.Local,
			Remote:  p.Remote,
			ServeOK: serveOK,
			LastRun: "never",
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
		}
		rep.Items = append(rep.Items, item)
	}
	return rep, nil
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
	return &st, nil
}
