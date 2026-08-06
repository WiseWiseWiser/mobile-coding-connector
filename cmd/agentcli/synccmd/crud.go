package synccmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultIgnore is the init default ignore list.
var DefaultIgnore = []string{
	"Name .DS_Store",
	"Name node_modules",
	"Name *.log",
	"Path tmp",
	"Path log",
}

// InitOpts configures InitPair.
type InitOpts struct {
	Name          string
	Local         string
	Remote        string
	Prefer        string // empty → "newer"
	LocalHostname string // empty → "remote-agent-<name>"
	RemoteUnison  string // empty → "/usr/local/bin/unison"
	Ignore        []string
	// Times/Auto/Batch: nil → true; non-nil → *value
	Times *bool
	Auto  *bool
	Batch *bool
}

// SetOpts is a partial update for SetPair. Pointer/nil fields are left unchanged.
// When IgnoreSet is true, Ignore replaces the pair's ignore list.
type SetOpts struct {
	Local         *string
	Remote        *string
	Prefer        *string
	LocalHostname *string
	RemoteUnison  *string
	Ignore        []string
	IgnoreSet     bool
	Times         *bool
	Auto          *bool
	Batch         *bool
}

// RmOpts controls RmPair profile cleanup.
type RmOpts struct {
	// PurgeProfile when true deletes {UnisonDir}/remote-agent-<name>.prf.
	PurgeProfile bool
	// UnisonDir is required when PurgeProfile is true.
	UnisonDir string
}

func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func absPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// InitPair creates a new pair with defaults. Errors if name already exists.
func InitPair(store *Store, opts InitOpts) (*Pair, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("init requires a pair name")
	}
	if opts.Local == "" || opts.Remote == "" {
		return nil, fmt.Errorf("init requires name, local, and remote paths")
	}

	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	for _, p := range cfg.Pairs {
		if p.Name == opts.Name {
			return nil, fmt.Errorf("pair %q already exists", opts.Name)
		}
	}

	local, err := absPath(opts.Local)
	if err != nil {
		return nil, fmt.Errorf("local path: %w", err)
	}
	remote, err := absPath(opts.Remote)
	if err != nil {
		return nil, fmt.Errorf("remote path: %w", err)
	}

	prefer := opts.Prefer
	if prefer == "" {
		prefer = "newer"
	}
	localHostname := opts.LocalHostname
	if localHostname == "" {
		localHostname = "remote-agent-" + opts.Name
	}
	remoteUnison := opts.RemoteUnison
	if remoteUnison == "" {
		remoteUnison = "/usr/local/bin/unison"
	}
	ignore := opts.Ignore
	if ignore == nil {
		ignore = append([]string(nil), DefaultIgnore...)
	}

	pair := Pair{
		Name:          opts.Name,
		Backend:       "unison",
		Local:         local,
		Remote:        remote,
		Prefer:        prefer,
		Ignore:        ignore,
		LocalHostname: localHostname,
		RemoteUnison:  remoteUnison,
		Times:         boolOrDefault(opts.Times, true),
		Auto:          boolOrDefault(opts.Auto, true),
		Batch:         boolOrDefault(opts.Batch, true),
	}
	cfg.Pairs = append(cfg.Pairs, pair)
	if err := store.Save(cfg); err != nil {
		return nil, err
	}
	return &pair, nil
}

// SetPair applies a partial update to an existing pair.
func SetPair(store *Store, name string, opts SetOpts) (*Pair, error) {
	if name == "" {
		return nil, fmt.Errorf("set requires a pair name")
	}
	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range cfg.Pairs {
		if cfg.Pairs[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("unknown pair: %s", name)
	}
	p := &cfg.Pairs[idx]

	if opts.Local != nil {
		local, err := absPath(*opts.Local)
		if err != nil {
			return nil, fmt.Errorf("local path: %w", err)
		}
		p.Local = local
	}
	if opts.Remote != nil {
		remote, err := absPath(*opts.Remote)
		if err != nil {
			return nil, fmt.Errorf("remote path: %w", err)
		}
		p.Remote = remote
	}
	if opts.Prefer != nil {
		p.Prefer = *opts.Prefer
	}
	if opts.LocalHostname != nil {
		p.LocalHostname = *opts.LocalHostname
	}
	if opts.RemoteUnison != nil {
		p.RemoteUnison = *opts.RemoteUnison
	}
	if opts.IgnoreSet {
		if opts.Ignore == nil {
			p.Ignore = []string{}
		} else {
			p.Ignore = append([]string(nil), opts.Ignore...)
		}
	}
	if opts.Times != nil {
		p.Times = *opts.Times
	}
	if opts.Auto != nil {
		p.Auto = *opts.Auto
	}
	if opts.Batch != nil {
		p.Batch = *opts.Batch
	}

	if err := store.Save(cfg); err != nil {
		return nil, err
	}
	out := *p
	return &out, nil
}

// RmPair removes a pair from the store; optionally purges the Unison profile.
func RmPair(store *Store, name string, opts RmOpts) error {
	if name == "" {
		return fmt.Errorf("rm requires a pair name")
	}
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	idx := -1
	for i := range cfg.Pairs {
		if cfg.Pairs[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("unknown pair: %s", name)
	}
	cfg.Pairs = append(cfg.Pairs[:idx], cfg.Pairs[idx+1:]...)
	if err := store.Save(cfg); err != nil {
		return err
	}
	if opts.PurgeProfile {
		if opts.UnisonDir == "" {
			return fmt.Errorf("purge profile requires UnisonDir")
		}
		path := filepath.Join(opts.UnisonDir, ProfileFileName(name))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("purge profile: %w", err)
		}
	}
	return nil
}
