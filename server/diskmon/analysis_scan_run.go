package diskmon

import (
	"context"
	"os"
	"sync"
	"time"
)

func runScan(ctx context.Context, cfg scanConfig) (scanResult, error) {
	if cfg.Budget <= 0 {
		cfg.Budget = 120 * time.Second
	}
	if cfg.Home == "" {
		cfg.Home, _ = os.UserHomeDir()
	}
	if cfg.Volume == "" {
		cfg.Volume = volumePrefix(cfg.Roots)
	}
	// Home is compared against walked paths, so it lives in the same space:
	// with the volume prefix dropped.
	cfg.Home = logicalPath(cfg.Home, cfg.Volume)

	workers := cfg.Workers
	if workers <= 0 {
		workers = defaultScanWorkers()
	}

	priorityList := priorityRootPaths(cfg.Home)
	acc := &scanAccumulator{
		cfg:          cfg,
		class:        newClassifier(cfg.Home, cfg.Volume),
		ctx:          ctx,
		deadline:     time.Now().Add(cfg.Budget),
		queue:        newScanQueue(),
		catBytes:     map[string]int64{},
		catReclaim:   map[string]int64{},
		links:        map[[2]uint64]*parkedLink{},
		visited:      map[string]bool{},
		skippedAt:    map[string]bool{},
		priorityList: priorityList,
	}

	stopProgress := make(chan struct{})
	var progress sync.WaitGroup
	progress.Add(1)
	go func() {
		defer progress.Done()
		acc.reportLoop(stopProgress)
	}()

	var walk sync.WaitGroup
	for i := 0; i < workers; i++ {
		walk.Add(1)
		go func() {
			defer walk.Done()
			acc.work()
		}()
	}

	acc.seed()
	walkDone := make(chan struct{})
	go func() {
		walk.Wait()
		close(walkDone)
	}()

	// A directory read cannot be interrupted, so one hung mount (a stale
	// autofs entry, a dying network volume) would otherwise keep this job
	// "running" forever. Past the deadline — or as soon as the user cancels —
	// the scan stops handing out work and waits only a short grace period for
	// the workers that are still inside a read.
	timer := time.NewTimer(cfg.Budget + scanGrace)
	defer timer.Stop()
	cancelled := false
	select {
	case <-walkDone:
	case <-ctx.Done():
		cancelled = true
		acc.queue.stop()
		waitWithin(walkDone, scanCancelGrace)
	case <-timer.C:
		select {
		case <-walkDone:
		default:
			acc.truncated.Store(true)
			acc.queue.stop()
			waitWithin(walkDone, scanGrace)
		}
	}
	close(stopProgress)
	progress.Wait()

	if cancelled {
		// Nothing is written for a cancelled scan, so there is no partial
		// result to hand back either.
		return scanResult{}, ctx.Err()
	}
	res := acc.finish()
	if err := ctx.Err(); err != nil {
		return res, err
	}
	return res, nil
}

// scanGrace is how long a walk may outlive its budget before the scan stops
// waiting for a stalled directory read and finishes with the partial result.
// scanCancelGrace is the shorter version used after an explicit cancel, where
// the result is discarded anyway.
const (
	scanGrace       = 15 * time.Second
	scanCancelGrace = 3 * time.Second
)

// waitWithin reports whether done fired before the timeout.
func waitWithin(done <-chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func defaultScanWorkers() int {
	workers := runtimeWorkers()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}
