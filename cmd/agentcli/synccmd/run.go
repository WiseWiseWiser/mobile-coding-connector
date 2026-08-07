package synccmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// ExecFunc runs a child process. Production default uses os/exec with cmd.Env
// only (never os.Setenv on the parent). Tests inject fakes.
type ExecFunc func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (exitCode int, err error)

// RunOpts configures BuildUnisonCmd / RunPair (injectable dirs, Exec, doctor hooks).
type RunOpts struct {
	StoreDir, UnisonDir, SSHConfigDir string
	Name                              string
	SkipDoctor                        bool
	Interactive                       bool
	LocalUnisonPath                   string // empty → "unison"
	Exec                              ExecFunc
	Stdout, Stderr                    io.Writer
	Context                           context.Context // nil → context.Background()
	// Doctor probes when !SkipDoctor (nil → product defaults; tests inject).
	LocalVersion  func() (string, error)
	RemoteVersion func() (string, error)
	ServeOK       func() error
	RemotePathOK  func(remote string) error
}

// RunResult is the outcome of RunPair after Exec returns.
type RunResult struct {
	ExitCode int
	Message  string
	Duration time.Duration // wall time of Exec; zero OK if unused
	Argv     []string      // optional: argv that was (or would be) executed
}

// BuildUnisonCmd builds Unison child argv + env for a named pair.
// Never mutates process environment; UNISONLOCALHOSTNAME is child-env only.
func BuildUnisonCmd(opts RunOpts) (argv []string, env []string, workdir string, err error) {
	if opts.Name == "" {
		return nil, nil, "", fmt.Errorf("run requires a pair name")
	}
	store := &Store{Dir: opts.StoreDir}
	pair, err := GetPair(store, opts.Name)
	if err != nil {
		return nil, nil, "", err
	}

	bin := opts.LocalUnisonPath
	if bin == "" {
		bin = "unison"
	}
	profile := "remote-agent-" + pair.Name

	// Full argv including binary as argv[0]; profile basename without .prf.
	argv = []string{bin, profile}
	// Non-interactive + pair.Batch → -batch; Interactive always omits -batch.
	if !opts.Interactive && pair.Batch {
		argv = append(argv, "-batch")
	}

	env = buildChildEnv(pair.LocalHostname, opts.UnisonDir)
	workdir = opts.UnisonDir
	return argv, env, workdir, nil
}

// buildChildEnv returns a full child env with UNISONLOCALHOSTNAME set and
// optional UNISON pointing at UnisonDir for profile discovery. Parent process
// env is never mutated.
func buildChildEnv(localHostname, unisonDir string) []string {
	base := os.Environ()
	// Filter keys we override so we do not duplicate.
	out := make([]string, 0, len(base)+2)
	for _, e := range base {
		if hasEnvKey(e, "UNISONLOCALHOSTNAME") || hasEnvKey(e, "UNISON") {
			continue
		}
		out = append(out, e)
	}
	out = append(out, "UNISONLOCALHOSTNAME="+localHostname)
	if unisonDir != "" {
		out = append(out, "UNISON="+unisonDir)
	}
	return out
}

func hasEnvKey(entry, key string) bool {
	if entry == key {
		return true
	}
	prefix := key + "="
	return len(entry) >= len(prefix) && entry[:len(prefix)] == prefix
}

// RunPair resolves a pair, optionally runs doctor, executes Unison via Exec,
// writes last-run state, and returns RunResult. Non-zero Exec exit → non-nil error
// but state is still written.
func RunPair(opts RunOpts) (RunResult, error) {
	var result RunResult

	if opts.Name == "" {
		return result, fmt.Errorf("run requires a pair name")
	}

	store := &Store{Dir: opts.StoreDir}
	if _, err := GetPair(store, opts.Name); err != nil {
		return result, err
	}

	if !opts.SkipDoctor {
		_, derr := Doctor(DoctorOpts{
			StoreDir:      opts.StoreDir,
			UnisonDir:     opts.UnisonDir,
			SSHConfigDir:  opts.SSHConfigDir,
			Name:          opts.Name,
			LocalVersion:  opts.LocalVersion,
			RemoteVersion: opts.RemoteVersion,
			ServeOK:       opts.ServeOK,
			RemotePathOK:  opts.RemotePathOK,
		})
		if derr != nil {
			// Abort before Exec and without writing state.
			return result, derr
		}
	}

	argv, env, workdir, err := BuildUnisonCmd(opts)
	if err != nil {
		return result, err
	}
	result.Argv = append([]string(nil), argv...)

	execFn := opts.Exec
	if execFn == nil {
		execFn = defaultExec
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	bin := argv[0]
	// Tee Unison output so we can parse the summary while streaming to the user.
	var capture bytes.Buffer
	stdout := io.Writer(&capture)
	stderr := io.Writer(&capture)
	if opts.Stdout != nil {
		stdout = io.MultiWriter(opts.Stdout, &capture)
	}
	if opts.Stderr != nil {
		stderr = io.MultiWriter(opts.Stderr, &capture)
	}

	// Pass full argv (including binary as argv[0]); defaultExec strips it.
	start := time.Now()
	exitCode, execErr := execFn(ctx, bin, argv, env, stdout, stderr)
	result.Duration = time.Since(start)
	result.ExitCode = exitCode

	outcome, detail, transferred, failed, skipped := parseUnisonOutcome(exitCode, capture.String())
	msg := detail
	if msg == "" {
		if exitCode != 0 {
			msg = fmt.Sprintf("exit code %d", exitCode)
		} else {
			msg = "ok"
		}
	}
	if execErr != nil && exitCode == 0 {
		msg = execErr.Error()
		outcome = OutcomeFailed
	}
	result.Message = msg

	// Write enriched state after Exec returns (including non-zero exit).
	if werr := writePairStateFull(opts.StoreDir, opts.Name, exitCode, msg, outcome, transferred, failed, skipped, result.Duration); werr != nil {
		if exitCode == 0 && execErr == nil {
			return result, fmt.Errorf("write state: %w", werr)
		}
	}

	if exitCode != 0 {
		if execErr != nil {
			return result, fmt.Errorf("unison exit code %d: %w", exitCode, execErr)
		}
		return result, fmt.Errorf("unison exit code %d", exitCode)
	}
	if execErr != nil {
		return result, execErr
	}
	_ = workdir
	return result, nil
}

// defaultExec runs the binary via os/exec with child-only env (no Setenv).
func defaultExec(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (int, error) {
	args := argv
	if len(argv) > 0 && argv[0] == name {
		args = argv[1:]
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if env != nil {
		cmd.Env = env
	}
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 1, err
}
