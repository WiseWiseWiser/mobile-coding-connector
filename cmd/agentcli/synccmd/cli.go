package synccmd

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// CLIOpts configures RunCLI (injectable dirs + writers + probe hooks + Exec; parallel-safe).
type CLIOpts struct {
	StoreDir     string
	UnisonDir    string
	SSHConfigDir string
	Stdout       io.Writer
	Stderr       io.Writer
	// Probe hooks for doctor/status/run (nil → product real defaults).
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	ServeOK       func() error
	RemotePathOK  func(remote string) error
	// Exec is the injectable Unison child runner for `unison run` (nil → default os/exec).
	Exec ExecFunc
	// Install hooks for `unison install` (nil → product defaults / stubs).
	LocalEnsure      func() (string, error)
	RemoteEnsure     func(targetPath string) (string, error)
	WhichLocal       func() (path, version string, err error)
	RemoteTargetPath string
}

// SyncUsage is printed for bare sync / -h / --help.
const SyncUsage = `Usage: remote-agent sync [subcommand]

Manage named local↔remote Unison sync pairs and regenerable profiles.

Subcommands:
  unison             Unison backend (init/add/list/show/set/rm)

Run 'remote-agent sync unison --help' for unison CRUD verbs.
`

// UnisonUsage is printed for bare unison / unison -h / --help.
const UnisonUsage = `Usage: remote-agent sync unison <command> [args...]

Unison pair store CRUD, profile emit, doctor, status, run, and install.

Commands:
  init|add <name> <local> <remote>   Create a pair (error if name exists)
  list                               List pair names
  show <name>                        Show one pair
  set <name> [flags]                 Partial-update a pair; regen profile
  rm <name> [--yes] [--purge-profile|--no-purge-profile]
                                     Remove pair (default: purge profile)
  doctor [<name>]                    Run readiness checks for a pair
  status [<name>]                    Show pair status and last-run state
  run <name> [--skip-doctor] [--interactive]
                                     Run Unison for a named pair
  install [--local|--remote|--both]  Ensure Unison binary (default: both)

Init/set flags:
  --prefer VALUE
  --local-hostname HOST
  --remote-unison PATH
  --local PATH / --remote PATH       (set only)
  --times / --no-times
  --auto / --no-auto
  --batch / --no-batch
  --ignore LINE                      (repeatable; on set replaces list)
`

func (o CLIOpts) stdout() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

func (o CLIOpts) stderr() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return os.Stderr
}

func (o CLIOpts) store() *Store {
	return &Store{Dir: o.StoreDir}
}

// RunCLI dispatches argv after the `sync` subcommand.
func RunCLI(args []string, opts CLIOpts) error {
	if args == nil {
		args = []string{}
	}

	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Fprint(opts.stdout(), SyncUsage)
		if !strings.HasSuffix(SyncUsage, "\n") {
			fmt.Fprintln(opts.stdout())
		}
		return nil
	}

	switch args[0] {
	case "help":
		fmt.Fprint(opts.stdout(), SyncUsage)
		return nil
	case "unison":
		return runUnison(args[1:], opts)
	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func isHelpToken(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}

func runUnison(args []string, opts CLIOpts) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Fprint(opts.stdout(), UnisonUsage)
		return nil
	}

	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "init", "add":
		return runInit(rest, opts)
	case "list":
		return runList(opts)
	case "show":
		return runShow(rest, opts)
	case "set":
		return runSet(rest, opts)
	case "rm":
		return runRm(rest, opts)
	case "doctor":
		return runDoctor(rest, opts)
	case "status":
		return runStatus(rest, opts)
	case "run":
		return runRun(rest, opts)
	case "install":
		return runInstall(rest, opts)
	default:
		return fmt.Errorf("unknown unison subcommand: %s", cmd)
	}
}

// runInstall handles `unison install [--local|--remote|--both]` (default both).
func runInstall(args []string, opts CLIOpts) error {
	scope := "both"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			scope = "local"
		case "--remote":
			scope = "remote"
		case "--both":
			scope = "both"
		case "-h", "--help", "help":
			fmt.Fprintln(opts.stdout(), "Usage: remote-agent sync unison install [--local|--remote|--both]")
			return nil
		default:
			return fmt.Errorf("unknown install flag: %s", args[i])
		}
	}
	report, err := Install(InstallOpts{
		Scope:            scope,
		LocalEnsure:      opts.LocalEnsure,
		RemoteEnsure:     opts.RemoteEnsure,
		WhichLocal:       opts.WhichLocal,
		RemoteTargetPath: opts.RemoteTargetPath,
		Stdout:           opts.stdout(),
		Stderr:           opts.stderr(),
	})
	for _, msg := range report.Messages {
		fmt.Fprintln(opts.stdout(), msg)
	}
	return err
}

// runRun handles `unison run <name> [--skip-doctor] [--interactive]`.
func runRun(args []string, opts CLIOpts) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("run requires a pair name")
	}
	name := args[0]
	skipDoctor := false
	interactive := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--skip-doctor":
			skipDoctor = true
		case "--interactive":
			interactive = true
		default:
			return fmt.Errorf("unknown run flag: %s", args[i])
		}
	}
	_, err := RunPair(RunOpts{
		StoreDir:      opts.StoreDir,
		UnisonDir:     opts.UnisonDir,
		SSHConfigDir:  opts.SSHConfigDir,
		Name:          name,
		SkipDoctor:    skipDoctor,
		Interactive:   interactive,
		Exec:          opts.Exec,
		Stdout:        opts.stdout(),
		Stderr:        opts.stderr(),
		LocalVersion:  opts.LocalVersion,
		RemoteVersion: opts.RemoteVersion,
		ServeOK:       opts.ServeOK,
		RemotePathOK:  opts.RemotePathOK,
	})
	return err
}

func runDoctor(args []string, opts CLIOpts) error {
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
	}
	report, err := Doctor(DoctorOpts{
		StoreDir:      opts.StoreDir,
		UnisonDir:     opts.UnisonDir,
		SSHConfigDir:  opts.SSHConfigDir,
		Name:          name,
		LocalVersion:  opts.LocalVersion,
		RemoteVersion: opts.RemoteVersion,
		ServeOK:       opts.ServeOK,
		RemotePathOK:  opts.RemotePathOK,
	})
	// Always print table-ish check lines when we have checks.
	if report.PairName != "" {
		fmt.Fprintf(opts.stdout(), "doctor: pair %s\n", report.PairName)
	} else {
		fmt.Fprintln(opts.stdout(), "doctor:")
	}
	for _, c := range report.Checks {
		status := "ok"
		if !c.OK {
			status = "fail"
		}
		fmt.Fprintf(opts.stdout(), "  %-16s %s  %s\n", c.Name, status, c.Detail)
	}
	if len(report.Checks) == 0 && err != nil {
		// Resolution failed before checks — still give a non-empty line.
		fmt.Fprintf(opts.stdout(), "  check fail: %v\n", err)
	}
	return err
}

func runStatus(args []string, opts CLIOpts) error {
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
	}
	report, err := Status(StatusOpts{
		StoreDir:     opts.StoreDir,
		UnisonDir:    opts.UnisonDir,
		SSHConfigDir: opts.SSHConfigDir,
		Name:         name,
		ServeOK:      opts.ServeOK,
	})
	if err != nil {
		return err
	}
	if len(report.Items) == 0 {
		fmt.Fprintln(opts.stdout(), "(no pairs)")
		return nil
	}
	for _, it := range report.Items {
		serve := "down"
		if it.ServeOK {
			serve = "up"
		}
		fmt.Fprintf(opts.stdout(), "%s  local=%s  remote=%s  serve=%s  lastRun=%s\n",
			it.Name, it.Local, it.Remote, serve, it.LastRun)
	}
	return nil
}

func runInit(args []string, opts CLIOpts) error {
	name, local, remote, flags, err := parseInitPositionals(args)
	if err != nil {
		return err
	}
	initOpts, err := parseInitFlags(flags)
	if err != nil {
		return err
	}
	initOpts.Name = name
	initOpts.Local = local
	initOpts.Remote = remote

	pair, err := InitPair(opts.store(), initOpts)
	if err != nil {
		return err
	}
	if _, err := WriteUnisonProfile(opts.UnisonDir, opts.SSHConfigDir, pair); err != nil {
		return err
	}
	fmt.Fprintf(opts.stdout(), "initialized pair %s\n", pair.Name)
	return nil
}

func parseInitPositionals(args []string) (name, local, remote string, flags []string, err error) {
	var pos []string
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			break
		}
		pos = append(pos, a)
		i++
	}
	if len(pos) < 3 {
		return "", "", "", nil, fmt.Errorf("init requires name, local, and remote paths")
	}
	return pos[0], pos[1], pos[2], args[i:], nil
}

func parseInitFlags(flags []string) (InitOpts, error) {
	var opts InitOpts
	for i := 0; i < len(flags); i++ {
		f := flags[i]
		needVal := func() (string, error) {
			if i+1 >= len(flags) {
				return "", fmt.Errorf("flag %s requires a value", f)
			}
			i++
			return flags[i], nil
		}
		switch f {
		case "--prefer":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.Prefer = v
		case "--local-hostname":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.LocalHostname = v
		case "--remote-unison":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.RemoteUnison = v
		case "--ignore":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.Ignore = append(opts.Ignore, v)
		case "--times":
			t := true
			opts.Times = &t
		case "--no-times":
			t := false
			opts.Times = &t
		case "--auto":
			t := true
			opts.Auto = &t
		case "--no-auto":
			t := false
			opts.Auto = &t
		case "--batch":
			t := true
			opts.Batch = &t
		case "--no-batch":
			t := false
			opts.Batch = &t
		default:
			return opts, fmt.Errorf("unknown init flag: %s", f)
		}
	}
	// When --ignore was used on init, opts.Ignore is non-nil (possibly empty slice
	// only if somehow set); nil means use defaults. append creates non-nil slice
	// only when at least one --ignore was seen — good.
	return opts, nil
}

func runList(opts CLIOpts) error {
	pairs, err := ListPairs(opts.store())
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		fmt.Fprintln(opts.stdout(), "(no pairs)")
		return nil
	}
	for _, p := range pairs {
		fmt.Fprintln(opts.stdout(), p.Name)
	}
	return nil
}

func runShow(args []string, opts CLIOpts) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("show requires a pair name")
	}
	name := args[0]
	p, err := GetPair(opts.store(), name)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.stdout(), "name: %s\n", p.Name)
	fmt.Fprintf(opts.stdout(), "backend: %s\n", p.Backend)
	fmt.Fprintf(opts.stdout(), "local: %s\n", p.Local)
	fmt.Fprintf(opts.stdout(), "remote: %s\n", p.Remote)
	fmt.Fprintf(opts.stdout(), "prefer: %s\n", p.Prefer)
	fmt.Fprintf(opts.stdout(), "localHostname: %s\n", p.LocalHostname)
	fmt.Fprintf(opts.stdout(), "remoteUnison: %s\n", p.RemoteUnison)
	fmt.Fprintf(opts.stdout(), "times: %t\n", p.Times)
	fmt.Fprintf(opts.stdout(), "auto: %t\n", p.Auto)
	fmt.Fprintf(opts.stdout(), "batch: %t\n", p.Batch)
	if len(p.Ignore) > 0 {
		fmt.Fprintln(opts.stdout(), "ignore:")
		for _, ign := range p.Ignore {
			fmt.Fprintf(opts.stdout(), "  %s\n", ign)
		}
	}
	return nil
}

func runSet(args []string, opts CLIOpts) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("set requires a pair name")
	}
	name := args[0]
	setOpts, err := parseSetFlags(args[1:])
	if err != nil {
		return err
	}
	pair, err := SetPair(opts.store(), name, setOpts)
	if err != nil {
		return err
	}
	if _, err := WriteUnisonProfile(opts.UnisonDir, opts.SSHConfigDir, pair); err != nil {
		return err
	}
	fmt.Fprintf(opts.stdout(), "updated pair %s\n", pair.Name)
	return nil
}

func parseSetFlags(flags []string) (SetOpts, error) {
	var opts SetOpts
	for i := 0; i < len(flags); i++ {
		f := flags[i]
		needVal := func() (string, error) {
			if i+1 >= len(flags) {
				return "", fmt.Errorf("flag %s requires a value", f)
			}
			i++
			return flags[i], nil
		}
		switch f {
		case "--local":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.Local = &v
		case "--remote":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.Remote = &v
		case "--prefer":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.Prefer = &v
		case "--local-hostname":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.LocalHostname = &v
		case "--remote-unison":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.RemoteUnison = &v
		case "--ignore":
			v, err := needVal()
			if err != nil {
				return opts, err
			}
			opts.IgnoreSet = true
			opts.Ignore = append(opts.Ignore, v)
		case "--times":
			t := true
			opts.Times = &t
		case "--no-times":
			t := false
			opts.Times = &t
		case "--auto":
			t := true
			opts.Auto = &t
		case "--no-auto":
			t := false
			opts.Auto = &t
		case "--batch":
			t := true
			opts.Batch = &t
		case "--no-batch":
			t := false
			opts.Batch = &t
		default:
			return opts, fmt.Errorf("unknown set flag: %s", f)
		}
	}
	return opts, nil
}

func runRm(args []string, opts CLIOpts) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("rm requires a pair name")
	}
	name := args[0]
	purge := true // default purge profile
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--yes", "-y":
			// accepted; no interactive confirm in library CLI
		case "--purge-profile":
			purge = true
		case "--no-purge-profile":
			purge = false
		default:
			return fmt.Errorf("unknown rm flag: %s", args[i])
		}
	}
	return RmPair(opts.store(), name, RmOpts{
		PurgeProfile: purge,
		UnisonDir:    opts.UnisonDir,
	})
}
