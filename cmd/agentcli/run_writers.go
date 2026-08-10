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

// osStdout returns the current process stdout (may be swapped by RunWithWriters).
func osStdout() *os.File {
	return os.Stdout
}

// RunWithWriters runs the CLI while directing process stdout/stderr to the
// given writers for the duration of the call. Writers may be *os.File or any
// io.Writer; non-file writers are wired through pipes. Nil writers keep the
// process defaults. Parallel-safe via package mutex.
func RunWithWriters(profile Profile, args []string, stdout, stderr io.Writer) error {
	runWritersMu.Lock()
	defer runWritersMu.Unlock()

	restore, err := redirectStdio(stdout, stderr)
	if err != nil {
		return err
	}
	defer restore()
	return Run(profile, args)
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
