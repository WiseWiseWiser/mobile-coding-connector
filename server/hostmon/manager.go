package hostmon

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// Manager samples the host and fsyncs each line. It stays stopped until Start
// or Boot sees the enabled file.
type Manager struct {
	mu       sync.Mutex
	dir      string
	interval time.Duration
	procRoot string
	now      func() time.Time
	collect  func(prev *counters) (Sample, *counters, error)

	running bool
	cancel  context.CancelFunc
	prev    *counters
}

// New returns a stopped manager that writes under dir.
func New(dir string) *Manager {
	return &Manager{
		dir:      dir,
		interval: Interval,
		procRoot: "/proc",
		now:      time.Now,
	}
}

var (
	defaultMu sync.Mutex
	def       *Manager
)

// Default is the process-wide monitor, rooted at <data dir>/hostmon.
func Default() *Manager {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if def == nil {
		def = New(filepath.Join(config.DataDir, "hostmon"))
	}
	return def
}

// Start begins sampling. It is a no-op when already running.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true
	m.prev = nil
	interval := m.interval
	m.mu.Unlock()

	go func() {
		// The first read is only a baseline. The next interval writes a delta.
		m.tick()
		m.loop(ctx, interval)
	}()
	return nil
}

// Stop ends sampling and keeps the log. It is a no-op when already stopped.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil
	}
	m.cancel()
	m.cancel = nil
	m.running = false
	m.prev = nil
	return nil
}

// SetAutoStart records whether the next server boot should sample.
// It does not start or stop the live sampler.
func (m *Manager) SetAutoStart(enabled bool) error {
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(m.dir, enabledName)
	if !enabled {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path, []byte("1\n"), 0644)
}

// AutoStart reports the persisted boot flag.
func (m *Manager) AutoStart() bool {
	_, err := os.Stat(filepath.Join(m.dir, enabledName))
	return err == nil
}

// Running reports whether the sampler goroutine is active.
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// LogPath is the current sample file.
func (m *Manager) LogPath() string {
	return filepath.Join(m.dir, logName)
}

// Status is safe to call while the sampler is stopped.
func (m *Manager) Status() Status {
	samples, _ := readSamples(m.dir, 1)
	status := Status{
		Running:     m.Running(),
		AutoStart:   m.AutoStart(),
		IntervalSec: int(m.interval / time.Second),
		LogPath:     m.LogPath(),
		LogBytes:    logBytes(m.dir),
	}
	if m.interval < time.Second {
		status.IntervalSec = 15
	}
	if len(samples) > 0 {
		latest := samples[len(samples)-1]
		status.Latest = &latest
	}
	return status
}

// Samples returns the last n persisted samples, oldest first.
func (m *Manager) Samples(n int) ([]Sample, error) {
	return readSamples(m.dir, n)
}

// Boot starts the sampler only when auto-start was enabled.
func (m *Manager) Boot() {
	if m.AutoStart() {
		_ = m.Start()
	}
}

// Boot starts the process-wide sampler when it was enabled.
func Boot() {
	Default().Boot()
}

func (m *Manager) loop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = Interval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick()
		}
	}
}

func (m *Manager) tick() {
	sample, next, err := m.sample()
	if err != nil {
		return
	}
	if sample.T == "" {
		m.mu.Lock()
		m.prev = next
		m.mu.Unlock()
		return
	}
	if err := appendSample(m.dir, sample); err != nil {
		return
	}
	m.mu.Lock()
	m.prev = next
	m.mu.Unlock()
}

func (m *Manager) sample() (Sample, *counters, error) {
	m.mu.Lock()
	prev := m.prev
	collect := m.collect
	root := m.procRoot
	nowFn := m.now
	m.mu.Unlock()
	if nowFn == nil {
		nowFn = time.Now
	}
	if collect != nil {
		return collect(prev)
	}
	cur, err := readProc(root, nowFn())
	if err != nil {
		return Sample{}, nil, err
	}
	if prev == nil {
		return Sample{}, &cur, nil
	}
	sample := deltaSample(*prev, cur, func(pid int) string {
		return readCmdline(root, pid)
	})
	return sample, &cur, nil
}
