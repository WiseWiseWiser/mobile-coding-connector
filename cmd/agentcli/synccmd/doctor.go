package synccmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DoctorOpts configures Doctor with injectable probe hooks (parallel-safe).
type DoctorOpts struct {
	StoreDir      string
	UnisonDir     string
	SSHConfigDir  string
	Name          string // empty → resolve defaultPair / sole pair
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	ServeOK       func() error // nil error = up
	RemotePathOK  func(remote string) error
}

// DoctorCheck is one named readiness row.
type DoctorCheck struct {
	Name   string
	OK     bool
	Detail string
}

// DoctorReport is the structured doctor outcome for one resolved pair.
type DoctorReport struct {
	PairName string
	Checks   []DoctorCheck
	AllOK    bool
}

// Doctor runs readiness checks for a named (or resolved) Unison pair.
// When the pair resolves but any check fails, returns a populated report
// (AllOK false) and a non-nil error so CLI can exit non-zero.
func Doctor(opts DoctorOpts) (DoctorReport, error) {
	store := &Store{Dir: opts.StoreDir}
	cfg, err := store.Load()
	if err != nil {
		// pairs-json load failed before resolve — still emit pairs-json row.
		rep := DoctorReport{
			Checks: []DoctorCheck{{
				Name:   "pairs-json",
				OK:     false,
				Detail: err.Error(),
			}},
			AllOK: false,
		}
		return rep, fmt.Errorf("doctor: pairs.json: %w", err)
	}

	name, err := resolvePairName(cfg, opts.Name)
	if err != nil {
		return DoctorReport{AllOK: false}, err
	}

	pair, err := GetPair(store, name)
	if err != nil {
		return DoctorReport{AllOK: false}, err
	}

	rep := DoctorReport{PairName: name, AllOK: true}
	var localVer, remoteVer string
	var localVerOK, remoteVerOK bool
	var serveErr error
	serveOK := true

	// pairs-json: store already loaded
	rep.Checks = append(rep.Checks, DoctorCheck{
		Name:   "pairs-json",
		OK:     true,
		Detail: "ok",
	})

	// local-version
	if opts.LocalVersion != nil {
		v, verr := opts.LocalVersion()
		if verr != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "local-version",
				OK:     false,
				Detail: verr.Error(),
			})
		} else {
			v = strings.TrimSpace(v)
			if v == "" {
				rep.Checks = append(rep.Checks, DoctorCheck{
					Name:   "local-version",
					OK:     false,
					Detail: "empty version",
				})
			} else {
				localVer = v
				localVerOK = true
				rep.Checks = append(rep.Checks, DoctorCheck{
					Name:   "local-version",
					OK:     true,
					Detail: v,
				})
			}
		}
	} else {
		// Production: PATH unison -version (tests inject LocalVersion).
		v, verr := defaultLocalVersion()
		if verr != nil || strings.TrimSpace(v) == "" {
			detail := "local unison version unavailable"
			if verr != nil {
				detail = verr.Error()
			}
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "local-version",
				OK:     false,
				Detail: detail,
			})
		} else {
			localVer = strings.TrimSpace(v)
			localVerOK = true
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "local-version",
				OK:     true,
				Detail: localVer,
			})
		}
	}

	// remote-version
	if opts.RemoteVersion != nil {
		v, verr := opts.RemoteVersion()
		if verr != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-version",
				OK:     false,
				Detail: verr.Error(),
			})
		} else {
			v = strings.TrimSpace(v)
			if v == "" {
				rep.Checks = append(rep.Checks, DoctorCheck{
					Name:   "remote-version",
					OK:     false,
					Detail: "empty version",
				})
			} else {
				remoteVer = v
				remoteVerOK = true
				rep.Checks = append(rep.Checks, DoctorCheck{
					Name:   "remote-version",
					OK:     true,
					Detail: v,
				})
			}
		}
	} else {
		// Production: ssh Host remote-agent + remoteUnison -version.
		v, verr := defaultRemoteVersion(opts.SSHConfigDir, pair.RemoteUnison)
		if verr != nil || strings.TrimSpace(v) == "" {
			detail := "remote unison version unavailable"
			if verr != nil {
				detail = verr.Error()
			}
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-version",
				OK:     false,
				Detail: detail,
			})
		} else {
			remoteVer = strings.TrimSpace(v)
			remoteVerOK = true
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-version",
				OK:     true,
				Detail: remoteVer,
			})
		}
	}

	// versions-match
	if localVerOK && remoteVerOK {
		if localVer == remoteVer {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "versions-match",
				OK:     true,
				Detail: localVer,
			})
		} else {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "versions-match",
				OK:     false,
				Detail: fmt.Sprintf("local %s != remote %s", localVer, remoteVer),
			})
		}
	} else {
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "versions-match",
			OK:     false,
			Detail: "cannot compare versions",
		})
	}

	// serve
	if opts.ServeOK != nil {
		serveErr = opts.ServeOK()
	} else {
		serveErr = defaultServeOK(opts.SSHConfigDir)
	}
	if serveErr != nil {
		serveOK = false
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "serve",
			OK:     false,
			Detail: serveErr.Error(),
		})
	} else {
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "serve",
			OK:     true,
			Detail: "up",
		})
	}

	// local-root
	if st, lerr := os.Stat(pair.Local); lerr != nil || !st.IsDir() {
		detail := "local root missing"
		if lerr != nil {
			detail = lerr.Error()
		} else {
			detail = "local root is not a directory"
		}
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "local-root",
			OK:     false,
			Detail: detail,
		})
	} else {
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "local-root",
			OK:     true,
			Detail: pair.Local,
		})
	}

	// remote-root
	if opts.RemotePathOK != nil {
		if rerr := opts.RemotePathOK(pair.Remote); rerr != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-root",
				OK:     false,
				Detail: rerr.Error(),
			})
		} else {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-root",
				OK:     true,
				Detail: pair.Remote,
			})
		}
	} else if !serveOK {
		// Default: when serve is down and no RemotePathOK hook, remote-root fails.
		detail := "serve down; cannot verify remote root"
		if serveErr != nil {
			detail = serveErr.Error()
		}
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "remote-root",
			OK:     false,
			Detail: detail,
		})
	} else {
		if rerr := defaultRemotePathOK(opts.SSHConfigDir, pair.Remote); rerr != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-root",
				OK:     false,
				Detail: rerr.Error(),
			})
		} else {
			rep.Checks = append(rep.Checks, DoctorCheck{
				Name:   "remote-root",
				OK:     true,
				Detail: pair.Remote,
			})
		}
	}

	// profile
	profPath := filepath.Join(opts.UnisonDir, ProfileFileName(name))
	if _, perr := os.Stat(profPath); perr != nil {
		detail := "profile missing"
		if perr != nil {
			detail = perr.Error()
		}
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "profile",
			OK:     false,
			Detail: detail,
		})
	} else {
		rep.Checks = append(rep.Checks, DoctorCheck{
			Name:   "profile",
			OK:     true,
			Detail: profPath,
		})
	}

	for _, c := range rep.Checks {
		if !c.OK {
			rep.AllOK = false
			break
		}
	}
	if !rep.AllOK {
		hint := ""
		if !serveOK {
			hint = "\nStart serve: remote-agent ssh --serve"
		}
		return rep, fmt.Errorf("doctor: one or more checks failed for pair %s%s", name, hint)
	}
	return rep, nil
}

// resolvePairName picks the pair: explicit → defaultPair → sole pair → error.
func resolvePairName(cfg *Config, name string) (string, error) {
	if name != "" {
		return name, nil
	}
	if cfg.DefaultPair != "" {
		return cfg.DefaultPair, nil
	}
	if len(cfg.Pairs) == 1 {
		return cfg.Pairs[0].Name, nil
	}
	return "", fmt.Errorf("pair name required")
}
