package diskmon

import "time"

// Env is injectable host I/O for tests.
type Env struct {
	Now        func() time.Time
	ReadMounts func() ([]mountEntry, error)
	LatestIO   func() (IO, bool)
}

func defaultEnv() Env {
	return Env{
		Now:        time.Now,
		ReadMounts: platformReadMounts,
		LatestIO:   latestHostIO,
	}
}
