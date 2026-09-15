package usageitems

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xhd2015/ai-critic/macosapp/codexusage"
	"github.com/xhd2015/ai-critic/macosapp/commandcodeusage"
	"github.com/xhd2015/ai-critic/macosapp/grokusage"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

// Snapshot is the rendered provider state of one item at one moment.
type Snapshot struct {
	Status    string
	Percent   string
	Body      string
	Detail    string
	UsageURL  string
	Error     string
	UpdatedAt string
}

// worker fetches and caches one item's provider usage.
type worker interface {
	EnsureFetch()
	FetchNow()
	Snapshot() Snapshot
	Start()
	Stop()
}

func newWorker(it Item) (worker, error) {
	switch it.Kind {
	case KindGrok:
		return &grokWorker{svc: grokusage.NewService()}, nil
	case KindCodex:
		return &codexWorker{svc: codexusage.NewService()}, nil
	case KindCommandCode:
		return &commandCodeWorker{svc: commandcodeusage.NewService(it.Home, it.APIURL)}, nil
	default:
		return nil, fmt.Errorf("unknown kind %q; known kinds: %s", it.Kind, strings.Join(ValidKinds(), ", "))
	}
}

// sameWorker reports whether an item change can reuse the existing worker.
func sameWorker(a, b Item) bool {
	return a.Kind == b.Kind && a.Home == b.Home && a.APIURL == b.APIURL
}

type grokWorker struct{ svc *grokusage.Service }

func (w *grokWorker) EnsureFetch() { w.svc.EnsureFetch() }
func (w *grokWorker) FetchNow()    { w.svc.FetchNow() }
func (w *grokWorker) Start()       { w.svc.Start() }
func (w *grokWorker) Stop()        { w.svc.Stop() }

func (w *grokWorker) Snapshot() Snapshot {
	resp := w.svc.Get()
	status := string(resp.Status)
	return Snapshot{
		Status:    status,
		Percent:   resp.WeeklyLimit,
		Body:      menubar.ComposeGrokBody(status, resp.WeeklyLimit, menubar.PeriodLabelForAPI(resp.Period), resp.ResetDisplay, resp.TimeLeft, resp.Error),
		Error:     resp.Error,
		UpdatedAt: resp.UpdatedAt,
	}
}

type codexWorker struct{ svc *codexusage.Service }

func (w *codexWorker) EnsureFetch() { w.svc.EnsureFetch() }
func (w *codexWorker) FetchNow()    { w.svc.FetchNow() }
func (w *codexWorker) Start()       { w.svc.Start() }
func (w *codexWorker) Stop()        { w.svc.Stop() }

func (w *codexWorker) Snapshot() Snapshot {
	resp := w.svc.Get()
	status := string(resp.Status)
	return Snapshot{
		Status:    status,
		Percent:   resp.MonthlyUsage,
		Body:      menubar.ComposeCodexBody(status, resp.MonthlyUsage, resp.CreditsUsed, resp.CreditsTotal, resp.ResetDisplay, resp.TimeLeft, resp.Error),
		Error:     resp.Error,
		UpdatedAt: resp.UpdatedAt,
	}
}

type commandCodeWorker struct{ svc *commandcodeusage.Service }

func (w *commandCodeWorker) EnsureFetch() { w.svc.EnsureFetch() }
func (w *commandCodeWorker) FetchNow()    { w.svc.FetchNow() }
func (w *commandCodeWorker) Start()       { w.svc.Start() }
func (w *commandCodeWorker) Stop()        { w.svc.Stop() }

func (w *commandCodeWorker) Snapshot() Snapshot {
	resp := w.svc.Get()
	return Snapshot{
		Status:    string(resp.Status),
		Percent:   commandcodeusage.TitleSuffix(resp),
		Body:      commandcodeusage.FormatBody(resp),
		Detail:    resp.Detail,
		UsageURL:  resp.UsageURL,
		Error:     resp.Error,
		UpdatedAt: resp.UpdatedAt,
	}
}

// funcWorker is a worker backed by an injected fetch function, for tests and
// for providers that need no background refresh loop.
type funcWorker struct {
	fetch func() (Snapshot, error)

	mu       sync.Mutex
	cached   Snapshot
	fetched  bool
	fetching bool
}

func newFuncWorker(fetch func() (Snapshot, error)) *funcWorker {
	return &funcWorker{fetch: fetch, cached: Snapshot{Status: StatusLoading}}
}

func (w *funcWorker) EnsureFetch() { w.FetchNow() }

func (w *funcWorker) FetchNow() {
	w.mu.Lock()
	if w.fetching {
		w.mu.Unlock()
		return
	}
	w.fetching = true
	w.mu.Unlock()

	snap, err := w.fetch()

	w.mu.Lock()
	defer w.mu.Unlock()
	w.fetching = false
	w.fetched = true
	if err != nil {
		w.cached = Snapshot{Status: StatusError, Error: err.Error()}
		return
	}
	w.cached = snap
}

func (w *funcWorker) Snapshot() Snapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.cached
}

func (w *funcWorker) Start() {}
func (w *funcWorker) Stop()  {}
