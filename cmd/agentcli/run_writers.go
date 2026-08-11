package agentcli

import (
	"io"
	"os"
	"sync"
)

// runWritersMu serializes package-level stdio swap for RunWithWriters.
// Existing Run uses fmt against os.Stdout/os.Stderr; parallel suites inject
// buffers via RunWithWriters under this mutex (and harness-level locks).
var runWritersMu sync.Mutex

// lockedWriter serializes writes so a buffer can safely receive both direct
// fmt.Fprint(stdout, …) from runCLI and concurrent io.Copy from the pipe
// that backs redirected os.Stdout/os.Stderr.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// osStdout returns the current process stdout (may be swapped by RunWithWriters).
func osStdout() *os.File {
	return os.Stdout
}

// RunWithWriters is the L2-injectable CLI entry for tests. It redirects
// process stdout/stderr to the given writers (for osStdout() / legacy
// fmt paths) and also passes those writers into the core CLI (for
// fmt.Fprint(stdout, …) paths such as help and event-bus). Writers may
// be *os.File or any io.Writer; non-file writers are wired through pipes.
// Nil writers keep the process defaults. Parallel-safe via package mutex.
func RunWithWriters(profile Profile, args []string, stdout, stderr io.Writer) error {
	runWritersMu.Lock()
	defer runWritersMu.Unlock()

	// Wrap non-file writers so direct runCLI writes and pipe-copy goroutines
	// never race on the same bytes.Buffer (undefined → flaky empty capture).
	if stdout != nil {
		if _, ok := stdout.(*os.File); !ok {
			stdout = &lockedWriter{w: stdout}
		}
	}
	if stderr != nil {
		if _, ok := stderr.(*os.File); !ok {
			stderr = &lockedWriter{w: stderr}
		}
	}

	restore, err := redirectStdio(stdout, stderr)
	if err != nil {
		return err
	}
	defer restore()

	out, errW := stdout, stderr
	if out == nil {
		out = os.Stdout
	}
	if errW == nil {
		errW = os.Stderr
	}
	return runCLI(profile, args, out, errW)
}

func redirectStdio(stdout, stderr io.Writer) (restore func(), err error) {
	oldOut, oldErr := os.Stdout, os.Stderr
	var cleanups []func()

	restore = func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	if stdout != nil {
		if f, ok := stdout.(*os.File); ok {
			os.Stdout = f
		} else {
			r, w, perr := os.Pipe()
			if perr != nil {
				return nil, perr
			}
			done := make(chan struct{})
			go func() {
				_, _ = io.Copy(stdout, r)
				_ = r.Close()
				close(done)
			}()
			os.Stdout = w
			cleanups = append(cleanups, func() {
				_ = w.Close()
				<-done
			})
		}
	}

	if stderr != nil {
		if f, ok := stderr.(*os.File); ok {
			os.Stderr = f
		} else {
			r, w, perr := os.Pipe()
			if perr != nil {
				restore()
				return nil, perr
			}
			done := make(chan struct{})
			go func() {
				_, _ = io.Copy(stderr, r)
				_ = r.Close()
				close(done)
			}()
			os.Stderr = w
			cleanups = append(cleanups, func() {
				_ = w.Close()
				<-done
			})
		}
	}

	return restore, nil
}
