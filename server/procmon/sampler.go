package procmon

import (
	"os"
	"runtime"
	"sync"
	"time"
)

// Env is injectable host I/O for tests.
type Env struct {
	Now      func() time.Time
	Sleep    func(time.Duration)
	PS       func() (string, error)
	Load     func() (Load, error)
	Memory   func() Memory
	NumCPU   int
	SelfPID  int
	Username string
}

func defaultEnv() Env {
	return Env{
		Now:      time.Now,
		Sleep:    time.Sleep,
		PS:       runPS,
		Load:     readLoad,
		Memory:   readMemory,
		NumCPU:   runtime.NumCPU(),
		SelfPID:  os.Getpid(),
		Username: currentUsername(),
	}
}

type sampler struct {
	env Env

	mu       sync.Mutex
	inflight bool
	wait     chan struct{}
	prev     []rawProc
	prevAt   time.Time
	cached   *Snapshot
	cachedAt time.Time
	lastErr  error
}

var defaultSampler = &sampler{env: defaultEnv()}

// Get returns a cached-or-fresh process table. Concurrent callers share one sample.
func Get() (*Snapshot, error) {
	return defaultSampler.get()
}

func (s *sampler) get() (*Snapshot, error) {
	s.mu.Lock()
	if s.cached != nil && s.env.Now().Sub(s.cachedAt) < minSampleInterval && s.lastErr == nil {
		out := s.cached
		s.mu.Unlock()
		return out, nil
	}
	if s.inflight {
		ch := s.wait
		s.mu.Unlock()
		<-ch
		s.mu.Lock()
		out, err := s.cached, s.lastErr
		s.mu.Unlock()
		return out, err
	}
	s.inflight = true
	s.wait = make(chan struct{})
	s.mu.Unlock()

	snap, err := s.compute()

	s.mu.Lock()
	s.cached = snap
	s.lastErr = err
	s.cachedAt = s.env.Now()
	s.inflight = false
	close(s.wait)
	s.mu.Unlock()
	return snap, err
}

func (s *sampler) compute() (*Snapshot, error) {
	curOut, err := s.env.PS()
	if err != nil {
		return nil, err
	}
	cur, err := parsePSTable(curOut)
	if err != nil {
		return nil, err
	}
	now := s.env.Now()

	s.mu.Lock()
	prev := s.prev
	prevAt := s.prevAt
	s.mu.Unlock()

	if len(prev) == 0 || now.Sub(prevAt) < 150*time.Millisecond {
		s.env.Sleep(firstDeltaWait)
		curOut, err = s.env.PS()
		if err != nil {
			return nil, err
		}
		next, err := parsePSTable(curOut)
		if err != nil {
			return nil, err
		}
		now2 := s.env.Now()
		dt := now2.Sub(now)
		if dt <= 0 {
			dt = firstDeltaWait
		}
		snap := s.finish(cur, next, dt, now2)
		s.mu.Lock()
		s.prev = next
		s.prevAt = now2
		s.mu.Unlock()
		return snap, nil
	}

	dt := now.Sub(prevAt)
	snap := s.finish(prev, cur, dt, now)
	s.mu.Lock()
	s.prev = cur
	s.prevAt = now
	s.mu.Unlock()
	return snap, nil
}

func (s *sampler) finish(prev, cur []rawProc, dt time.Duration, at time.Time) *Snapshot {
	load, _ := s.env.Load()
	mem := s.env.Memory()
	return buildSnapshot(prev, cur, dt, at, mem, load, s.env.SelfPID, s.env.Username, s.env.NumCPU)
}
