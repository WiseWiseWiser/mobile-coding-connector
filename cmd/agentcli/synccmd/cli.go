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
	// ColorMode controls ANSI on human output (default Auto: TTY + NO_COLOR).
	ColorMode ColorMode
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
const SyncUsage = `Usage: remote-agent sync <command> [args...]

Local↔remote directory sync (Unison backend).

Commands:
  list                 List configured pairs
  run [<name>]         Run Unison for a pair (default pair if omitted)
  doctor [<name>]      Readiness checks
  status [<name>] [--check]  Sync outcome + last sync (optional live check)
  unison                     Unison backend (init/set/rm/install/…)

Prerequisite: remote-agent ssh --serve in another terminal.

Run 'remote-agent sync unison --help' for init/set/rm/install and flags.
`

// UnisonUsage is printed for bare unison / unison -h / --help.
const UnisonUsage = `Usage: remote-agent sync unison <command> [args...]

Unison pair store, profile emit, doctor, status, run, and install.

Commands:
  init|add <name> <local> <remote>   Create a pair (error if name exists)
  list                               List pair names
  show <name>                        Show one pair
  set <name> [flags]                 Partial-update a pair; regen profile
  rm <name> [--yes] [--purge-profile|--no-purge-profile]
                                     Remove pair (default: purge profile)
  doctor [<name>]                    Run readiness checks for a pair
  status [<name>] [--check]          Serve + SYNC + LAST SYNC + DETAIL
  run [<name>] [--skip-doctor] [--interactive]
                                     Run Unison (default pair if name omitted)
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

Run 'remote-agent sync unison <command> --help' for command-specific help.
`

const (
	initHelp = `Usage: remote-agent sync unison init|add <name> <local> <remote> [flags]

Create a named local↔remote Unison pair and emit ~/.unison/remote-agent-<name>.prf.

Flags:
  --prefer VALUE
  --local-hostname HOST
  --remote-unison PATH
  --times / --no-times
  --auto / --no-auto
  --batch / --no-batch
  --ignore LINE          (repeatable)
  -h, --help
`

	runHelp = `Usage: remote-agent sync unison run [<name>] [--skip-doctor] [--interactive]
       remote-agent sync run [<name>] …

Run Unison for a pair. Name may be omitted when defaultPair is set or only one pair exists.

Flags:
  --skip-doctor    skip readiness checks
  --interactive    non-batch Unison (may prompt)
  -h, --help

Prerequisite: remote-agent ssh --serve
`

	doctorHelp = `Usage: remote-agent sync unison doctor [<name>]
       remote-agent sync doctor [<name>]

Check local/remote Unison versions, ssh serve, roots, and profile.

  -h, --help
`

	statusHelp = `Usage: remote-agent sync unison status [<name>] [--check]
       remote-agent sync status [<name>] [--check]

Show serve health, sync outcome, and last successful sync time.

  --check     run Unison now (may transfer; serve required; can be slow)
  -h, --help

SYNC values: synced (nothing to do), ok (last run exit 0 with transfers),
failed, never. No fake progress percentages.
`

	listHelp = `Usage: remote-agent sync unison list
       remote-agent sync list

List configured sync pairs.

  -h, --help
`

	installHelp = `Usage: remote-agent sync unison install [--local|--remote|--both]

Ensure Unison is available (default: both). Preferred version: ` + PreferredUnisonVersion + `.

  -h, --help
`

	showHelp = `Usage: remote-agent sync unison show <name>

Show one pair definition.

  -h, --help
`

	setHelp = `Usage: remote-agent sync unison set <name> [flags]

Partial-update a pair and regenerate its Unison profile.

Flags:
  --local PATH / --remote PATH
  --prefer VALUE
  --local-hostname HOST
  --remote-unison PATH
  --times / --no-times / --auto / --no-auto / --batch / --no-batch
  --ignore LINE          (repeatable; replaces ignore list)
  -h, --help
`

	rmHelp = `Usage: remote-agent sync unison rm <name> [--yes] [--purge-profile|--no-purge-profile]

Remove a pair from the store. Default: purge generated profile.

  -h, --help
`
)

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

func (o CLIOpts) style() colorStyle {
	return newColorStyle(o.ColorMode, o.stdout())
}

// RunCLI dispatches argv after the `sync` subcommand.
func RunCLI(args []string, opts CLIOpts) error {
	if args == nil {
		args = []string{}
	}

	if len(args) == 0 || isHelpToken(args[0]) {
		printUsage(opts.stdout(), SyncUsage)
		return nil
	}

	switch args[0] {
	case "help":
		printUsage(opts.stdout(), SyncUsage)
		return nil
	case "unison":
		return runUnison(args[1:], opts)
	// Short aliases (go-best-practice: short path for primary verbs).
	case "list":
		return runList(args[1:], opts)
	case "run":
		return runRun(args[1:], opts)
	case "doctor":
		return runDoctor(args[1:], opts)
	case "status":
		return runStatus(args[1:], opts)
	default:
		return fmt.Errorf("unknown subcommand: %s\nRun 'remote-agent sync --help' for usage", args[0])
	}
}

func printUsage(w io.Writer, s string) {
	fmt.Fprint(w, s)
	if !strings.HasSuffix(s, "\n") {
		fmt.Fprintln(w)
	}
}

func isHelpToken(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}

func runUnison(args []string, opts CLIOpts) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printUsage(opts.stdout(), UnisonUsage)
		return nil
	}

	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "init", "add":
		return runInit(rest, opts)
	case "list":
		return runList(rest, opts)
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
		return fmt.Errorf("unknown unison subcommand: %s\nRun 'remote-agent sync unison --help' for usage", cmd)
	}
}

// runInstall handles `unison install [--local|--remote|--both]` (default both).
func runInstall(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), installHelp)
		return nil
	}
	scope := "both"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			scope = "local"
		case "--remote":
			scope = "remote"
		case "--both":
			scope = "both"
		default:
			return fmt.Errorf("unknown install flag: %s\nRun 'remote-agent sync unison install --help'", args[i])
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

// runRun handles `run [<name>] [--skip-doctor] [--interactive]`.
func runRun(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), runHelp)
		return nil
	}
	name := ""
	skipDoctor := false
	interactive := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			switch a {
			case "--skip-doctor":
				skipDoctor = true
			case "--interactive":
				interactive = true
			default:
				return fmt.Errorf("unknown run flag: %s\nRun 'remote-agent sync run --help'", a)
			}
			continue
		}
		if name == "" {
			name = a
			continue
		}
		return fmt.Errorf("run: unexpected argument %q", a)
	}
	// Resolve empty name via store (defaultPair / sole pair).
	if name == "" {
		cfg, err := opts.store().Load()
		if err != nil {
			return err
		}
		resolved, rerr := resolvePairName(cfg, "")
		if rerr != nil {
			return fmt.Errorf("run requires a pair name (or set defaultPair)\nRun 'remote-agent sync list'")
		}
		name = resolved
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
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), doctorHelp)
		return nil
	}
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
	st := opts.style()
	if report.PairName != "" {
		fmt.Fprintf(opts.stdout(), "doctor: pair %s\n", report.PairName)
	} else {
		fmt.Fprintln(opts.stdout(), "doctor:")
	}
	for _, c := range report.Checks {
		status := st.green("ok")
		if !c.OK {
			status = st.red("fail")
		}
		fmt.Fprintf(opts.stdout(), "  %-16s %s  %s\n", c.Name, status, c.Detail)
	}
	if len(report.Checks) == 0 && err != nil {
		fmt.Fprintf(opts.stdout(), "  check fail: %v\n", err)
	}
	if err == nil && report.AllOK {
		fmt.Fprintln(opts.stdout(), st.green("doctor: all checks passed"))
	}
	return err
}

func runStatus(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), statusHelp)
		return nil
	}
	name := ""
	check := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--check" {
			check = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			return fmt.Errorf("unknown status flag: %s\nRun 'remote-agent sync status --help'", a)
		}
		if name == "" {
			name = a
			continue
		}
		return fmt.Errorf("status: unexpected argument %q", a)
	}
	if check {
		fmt.Fprintln(opts.stderr(), "checking… (may transfer; serve required)")
	}
	report, err := Status(StatusOpts{
		StoreDir:      opts.StoreDir,
		UnisonDir:     opts.UnisonDir,
		SSHConfigDir:  opts.SSHConfigDir,
		Name:          name,
		ServeOK:       opts.ServeOK,
		Check:         check,
		Exec:          opts.Exec,
		LocalVersion:  opts.LocalVersion,
		RemoteVersion: opts.RemoteVersion,
		RemotePathOK:  opts.RemotePathOK,
		Stdout:        opts.stdout(),
		Stderr:        opts.stderr(),
	})
	// Always print table when we have items, even if check partially failed.
	st := opts.style()
	if !reportHasItems(report) && err != nil {
		return err
	}
	if len(report.Items) == 0 {
		fmt.Fprintln(opts.stdout(), "(no pairs)")
		return err
	}
	if !reportServeAllUp(report) {
		fmt.Fprintln(opts.stderr(), st.yellow("warning: serve down; SYNC reflects last run only"))
	}
	// Pad plain fields first, then color — ANSI escapes must not participate
	// in width calculation or columns misalign.
	fmt.Fprintf(opts.stdout(), "  %-16s %-6s %-8s %-22s %s\n", "NAME", "SERVE", "SYNC", "LAST SYNC", "DETAIL")
	for _, it := range report.Items {
		servePlain := padRight("down", 6)
		serve := st.red(servePlain)
		if it.ServeOK {
			servePlain = padRight("up", 6)
			serve = st.green(servePlain)
		}
		syncPlain := padRight(it.Sync, 8)
		var syncCol string
		switch it.Sync {
		case OutcomeSynced, OutcomePropagated:
			syncCol = st.green(syncPlain)
		case OutcomeFailed:
			syncCol = st.red(syncPlain)
		default:
			syncCol = st.gray(syncPlain)
		}
		fmt.Fprintf(opts.stdout(), "  %-16s %s %s %-22s %s\n",
			it.Name, serve, syncCol, it.LastSync, it.Detail)
	}
	fmt.Fprintf(opts.stdout(), "\n%s\n", st.gray(fmt.Sprintf("%d pair(s)", len(report.Items))))
	return err
}

// padRight pads s with spaces to width n (rune-aware for plain ASCII labels).
func padRight(s string, n int) string {
	if n <= 0 {
		return s
	}
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func reportHasItems(r StatusReport) bool {
	return len(r.Items) > 0
}

func reportServeAllUp(r StatusReport) bool {
	for _, it := range r.Items {
		if !it.ServeOK {
			return false
		}
	}
	return len(r.Items) > 0
}

func runInit(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), initHelp)
		return nil
	}
	name, local, remote, flags, err := parseInitPositionals(args)
	if err != nil {
		return fmt.Errorf("%w\nRun 'remote-agent sync unison init --help'", err)
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
	return opts, nil
}

func runList(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), listHelp)
		return nil
	}
	pairs, err := ListPairs(opts.store())
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		fmt.Fprintln(opts.stdout(), "(no pairs)")
		return nil
	}
	st := opts.style()
	fmt.Fprintf(opts.stdout(), "  %-16s %-40s %s\n", "NAME", "LOCAL", "REMOTE")
	for _, p := range pairs {
		local := p.Local
		if len(local) > 40 {
			local = "…" + local[len(local)-39:]
		}
		fmt.Fprintf(opts.stdout(), "  %-16s %-40s %s\n", p.Name, local, p.Remote)
	}
	fmt.Fprintf(opts.stdout(), "\n%s\n", st.gray(fmt.Sprintf("%d pair(s)", len(pairs))))
	return nil
}

func runShow(args []string, opts CLIOpts) error {
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), showHelp)
		return nil
	}
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("show requires a pair name\nRun 'remote-agent sync unison show --help'")
	}
	name := args[0]
	p, err := GetPair(opts.store(), name)
	if err != nil {
		return fmt.Errorf("%w\nRun 'remote-agent sync list' to see pairs", err)
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
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), setHelp)
		return nil
	}
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("set requires a pair name\nRun 'remote-agent sync unison set --help'")
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
	if hasHelpFlag(args) {
		printUsage(opts.stdout(), rmHelp)
		return nil
	}
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("rm requires a pair name\nRun 'remote-agent sync unison rm --help'")
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

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if isHelpToken(a) {
			return true
		}
	}
	return false
}
