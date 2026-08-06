// Package sshcmd implements the injectable library API for `remote-agent ssh`
// (P1): argv parse modes, optional user@host strip, session gate, help, and
// --serve exclusive rules — without real network or OpenSSH.
package sshcmd

import (
	"errors"
	"fmt"
	"io"
	"regexp"
)

// Mode classifies remote-agent ssh argv after the `ssh` subcommand.
type Mode string

const (
	ModeHelp    Mode = "help"
	ModeServe   Mode = "serve"
	ModeLogin   Mode = "login"
	ModeCommand Mode = "command"
)

// Usage is printed for -h / --help (trailing newline required).
const Usage = `Usage: remote-agent ssh --serve [--quiet]
       remote-agent ssh [user@host] [command [args...]]

  --serve     Start local relay + remote ad-hoc SSH; blocks until Ctrl-C.
              Run client commands in another terminal while serve is up.
  --quiet     With --serve: suppress the ready/stop banner.
  -h, --help  Show this help
`

// ErrNoActiveTunnel is returned when client modes lack an Alive session.
const ErrNoActiveTunnel = "no active SSH tunnel; run 'remote-agent ssh --serve' first"

// destMatcher matches strict OpenSSH-shaped user@host only (no path separators).
var destMatcher = regexp.MustCompile(`^[^@/\s]+@[^@/\s]+$`)

// ParseResult is the classified argv for remote-agent ssh.
type ParseResult struct {
	Mode       Mode
	Dest       string
	RemoteArgv []string
	// Quiet is set when --quiet appears (meaningful with --serve).
	Quiet bool
}

// Session is tunnel session metadata loaded from a SessionStore.
type Session struct {
	LocalPort int
	User      string
	ConfigDir string
	ServePID  int
	ProfileID string
	Alive     bool
}

// SessionStore loads active tunnel session metadata by profile id.
type SessionStore interface {
	Load(profileID string) (*Session, error)
}

// ServeOpts configures ServeStarter.Start.
type ServeOpts struct {
	ProfileID string
	Quiet     bool
	// Stdout receives serve ready/stop banners when non-nil and not Quiet.
	Stdout io.Writer
}

// ServeStarter starts the local serve side (`--serve` only).
type ServeStarter interface {
	Start(opts ServeOpts) error
}

// RunnerOpts is passed to SSHRunner.Run.
type RunnerOpts struct {
	ProfileID string
}

// SSHRunner runs interactive login or a remote command through a live session.
type SSHRunner interface {
	Run(sess *Session, remoteArgv []string, opts RunnerOpts) error
}

// Options configures sshcmd.Run with injectable dependencies (parallel-safe).
type Options struct {
	Args      []string
	ProfileID string
	Store     SessionStore
	Serve     ServeStarter
	Runner    SSHRunner
	Stdout    io.Writer
	Stderr    io.Writer
}

// Parse classifies argv into help | serve | login | command and strips optional dest.
func Parse(args []string) (*ParseResult, error) {
	if args == nil {
		args = []string{}
	}

	// Help short-circuits when the only arg is a help flag.
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		return &ParseResult{Mode: ModeHelp}, nil
	}

	// Scan for --serve / --quiet and remaining tokens.
	hasServe := false
	quiet := false
	var rest []string
	for _, a := range args {
		switch a {
		case "--serve":
			hasServe = true
			continue
		case "--quiet", "-q":
			quiet = true
			continue
		}
		rest = append(rest, a)
	}

	if hasServe {
		if len(rest) > 0 {
			return nil, errors.New("--serve cannot be combined with a remote command")
		}
		return &ParseResult{Mode: ModeServe, Quiet: quiet}, nil
	}
	if quiet {
		return nil, errors.New("--quiet is only valid with --serve")
	}

	dest := ""
	remote := rest
	if len(rest) > 0 && isDest(rest[0]) {
		dest = rest[0]
		remote = append([]string(nil), rest[1:]...)
	} else if remote == nil {
		remote = []string{}
	} else {
		remote = append([]string(nil), remote...)
	}

	if len(remote) == 0 {
		return &ParseResult{
			Mode:       ModeLogin,
			Dest:       dest,
			RemoteArgv: remote,
		}, nil
	}
	return &ParseResult{
		Mode:       ModeCommand,
		Dest:       dest,
		RemoteArgv: remote,
	}, nil
}

func isDest(token string) bool {
	return destMatcher.MatchString(token)
}

// Run executes the classified mode with injected deps (no process env/cwd).
func Run(opts Options) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	parsed, err := Parse(opts.Args)
	if err != nil {
		return err
	}

	switch parsed.Mode {
	case ModeHelp:
		_, err := io.WriteString(stdout, Usage)
		return err

	case ModeServe:
		if opts.Serve == nil {
			return errors.New("serve starter not configured")
		}
		return opts.Serve.Start(ServeOpts{
			ProfileID: opts.ProfileID,
			Quiet:     parsed.Quiet,
			Stdout:    stdout,
		})

	case ModeLogin, ModeCommand:
		if opts.Store == nil {
			return errors.New("session store not configured")
		}
		sess, err := opts.Store.Load(opts.ProfileID)
		if err != nil {
			return err
		}
		if sess == nil || !sess.Alive {
			return errors.New(ErrNoActiveTunnel)
		}
		if opts.Runner == nil {
			return errors.New("ssh runner not configured")
		}
		remote := parsed.RemoteArgv
		if remote == nil {
			remote = []string{}
		}
		return opts.Runner.Run(sess, remote, RunnerOpts{ProfileID: opts.ProfileID})

	default:
		return fmt.Errorf("unknown mode %q", parsed.Mode)
	}
}
