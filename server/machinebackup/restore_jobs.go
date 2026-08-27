package machinebackup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	RestorePhaseClassify = "classify"
	RestorePhaseApply    = "apply"
	RestorePhaseReady    = "ready"
)

// RestoreJobView is the JSON snapshot for restore poll clients.
type RestoreJobView struct {
	ID         string `json:"job_id"`
	Status     string `json:"status"`
	Phase      string `json:"phase,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
	Skip       int    `json:"skip_identical,omitempty"`
	Update     int    `json:"update,omitempty"`
	Create     int    `json:"create,omitempty"`
	Total      int    `json:"total_entries,omitempty"`
	Error      string `json:"error,omitempty"`
}

// RestoreJobList is GET /restore/jobs.
type RestoreJobList struct {
	Active *RestoreJobView `json:"active"`
	Last   *RestoreJobView `json:"last"`
}

type restoreJob struct {
	id         string
	status     string
	phase      string
	startedAt  time.Time
	updatedAt  time.Time
	finishedAt time.Time
	progressAt time.Time
	dryRun     bool
	exclude    []string
	include    []string
	archive    []byte
	skip       int
	update     int
	create     int
	total      int
	errMsg     string
	stopStatus string
	stopErr    string
	ctx        context.Context
	cancel     context.CancelFunc
}

func (j *restoreJob) view() RestoreJobView {
	return RestoreJobView{
		ID:         j.id,
		Status:     j.status,
		Phase:      j.phase,
		StartedAt:  rfc3339(j.startedAt),
		UpdatedAt:  rfc3339(j.updatedAt),
		FinishedAt: rfc3339(j.finishedAt),
		DryRun:     j.dryRun,
		Skip:       j.skip,
		Update:     j.update,
		Create:     j.create,
		Total:      j.total,
		Error:      j.errMsg,
	}
}

type restoreJobManager struct {
	homeFn         func() string
	heartbeatEvery time.Duration
	staleAfter     time.Duration
	now            func() time.Time
	beforeApply    func(*restoreJob)

	mu     sync.Mutex
	active *restoreJob
	last   *restoreJob
}

func newRestoreJobManager(homeFn func() string) *restoreJobManager {
	return &restoreJobManager{
		homeFn:         homeFn,
		heartbeatEvery: defaultJobHeartbeat,
		staleAfter:     defaultJobStale,
		now:            time.Now,
	}
}

func (m *restoreJobManager) start(archive []byte, dryRun bool, exclude, include []string) (RestoreJobView, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && (m.active.status == JobQueued || m.active.status == JobRunning) {
		return m.active.view(), false
	}
	now := m.now()
	ctx, cancel := context.WithCancel(context.Background())
	job := &restoreJob{
		id:         newJobID(),
		status:     JobQueued,
		startedAt:  now,
		updatedAt:  now,
		progressAt: now,
		dryRun:     dryRun,
		exclude:    exclude,
		include:    include,
		archive:    archive,
		ctx:        ctx,
		cancel:     cancel,
	}
	m.active = job
	go m.run(job)
	return job.view(), true
}

func (m *restoreJobManager) get(id string) (RestoreJobView, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && m.active.id == id {
		return m.active.view(), true
	}
	if m.last != nil && m.last.id == id {
		return m.last.view(), true
	}
	return RestoreJobView{}, false
}

func (m *restoreJobManager) list() RestoreJobList {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out RestoreJobList
	if m.active != nil {
		v := m.active.view()
		out.Active = &v
	}
	if m.last != nil {
		v := m.last.view()
		out.Last = &v
	}
	return out
}

func (m *restoreJobManager) cancel(id string) (RestoreJobView, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.findLocked(id)
	if job == nil {
		return RestoreJobView{}, false, nil
	}
	if job.status != JobQueued && job.status != JobRunning {
		return job.view(), true, fmt.Errorf("job is %s", job.status)
	}
	m.requestStopLocked(job, JobCanceled, "canceled")
	return job.view(), true, nil
}

func (m *restoreJobManager) findLocked(id string) *restoreJob {
	if m.active != nil && m.active.id == id {
		return m.active
	}
	if m.last != nil && m.last.id == id {
		return m.last
	}
	return nil
}

func (m *restoreJobManager) requestStopLocked(job *restoreJob, status, msg string) {
	if job.stopStatus != "" {
		return
	}
	job.stopStatus = status
	job.stopErr = msg
	if job.cancel != nil {
		job.cancel()
	}
}

func (m *restoreJobManager) isStopped(job *restoreJob) (bool, string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.ctx.Err() == nil {
		return false, "", ""
	}
	if job.stopStatus != "" {
		return true, job.stopStatus, job.stopErr
	}
	return true, JobCanceled, "canceled"
}

func (m *restoreJobManager) run(job *restoreJob) {
	defer job.cancel()
	m.startHeartbeat(job)
	if m.beforeApply != nil {
		m.beforeApply(job)
	}
	if stopped, st, msg := m.isStopped(job); stopped {
		m.finish(job, st, msg)
		return
	}
	m.setRunning(job, RestorePhaseClassify)
	home := strings.TrimSpace(m.homeFn())
	var plan *MachineRestorePlan
	var err error
	if job.dryRun {
		plan, err = BuildRestorePlan(home, bytes.NewReader(job.archive), job.exclude, job.include)
	} else {
		m.setPhase(job, RestorePhaseApply)
		plan, err = ApplyRestore(home, bytes.NewReader(job.archive), job.exclude, job.include)
	}
	if err != nil {
		if stopped, st, msg := m.isStopped(job); stopped {
			m.finish(job, st, msg)
			return
		}
		m.finish(job, JobError, err.Error())
		return
	}
	m.setCounts(job, plan)
	m.setPhase(job, RestorePhaseReady)
	m.finish(job, JobDone, "")
}

func countsFromRestorePlan(plan *MachineRestorePlan) (skip, update, create, total int) {
	if plan == nil {
		return
	}
	for _, e := range plan.Entries {
		total++
		switch e.Action {
		case "skip":
			skip++
		case "update":
			update++
		case "create":
			create++
		}
	}
	return
}

func (m *restoreJobManager) startHeartbeat(job *restoreJob) {
	every := m.heartbeatEvery
	if every <= 0 {
		every = defaultJobHeartbeat
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-job.ctx.Done():
				return
			case <-t.C:
				m.mu.Lock()
				if job.status == JobRunning {
					job.updatedAt = m.now()
					if m.staleAfter > 0 && m.now().Sub(job.progressAt) > m.staleAfter {
						m.requestStopLocked(job, JobError, "restore job stale: no progress")
					}
				}
				m.mu.Unlock()
			}
		}
	}()
}

func (m *restoreJobManager) setRunning(job *restoreJob, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	job.status = JobRunning
	job.phase = phase
	job.updatedAt = now
	job.progressAt = now
}

func (m *restoreJobManager) setPhase(job *restoreJob, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	job.phase = phase
	job.updatedAt = now
	job.progressAt = now
}

func (m *restoreJobManager) setCounts(job *restoreJob, plan *MachineRestorePlan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if plan != nil {
		job.skip, job.update, job.create, job.total = countsFromRestorePlan(plan)
	}
	now := m.now()
	job.updatedAt = now
	job.progressAt = now
}

func (m *restoreJobManager) finish(job *restoreJob, status, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.status == JobDone || job.status == JobError || job.status == JobCanceled {
		return
	}
	now := m.now()
	job.status = status
	job.errMsg = errMsg
	job.finishedAt = now
	job.updatedAt = now
	job.archive = nil
	if m.active != nil && m.active.id == job.id {
		m.last = job
		m.active = nil
	}
}

func registerRestoreJobRoutes(mux *http.ServeMux, m *restoreJobManager) {
	mux.HandleFunc("/api/remote-agent/machine/restore/jobs", func(w http.ResponseWriter, r *http.Request) {
		m.handleCollection(w, r)
	})
	mux.HandleFunc("/api/remote-agent/machine/restore/jobs/", func(w http.ResponseWriter, r *http.Request) {
		m.handleItem(w, r)
	})
}

func (m *restoreJobManager) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, m.list())
	case http.MethodPost:
		dryRun, err := parseDryRunQuery(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		exclude, include := parsePathRulesQuery(r)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("read archive: %v", err))
			return
		}
		if len(body) == 0 {
			writeJSONError(w, http.StatusBadRequest, "archive body is required")
			return
		}
		view, started := m.start(body, dryRun, exclude, include)
		if !started {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  "restore job already running",
				"job_id": view.ID,
				"job":    view,
			})
			return
		}
		writeJSON(w, http.StatusAccepted, view)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (m *restoreJobManager) handleItem(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/remote-agent/machine/restore/jobs/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		m.handleCollection(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		view, ok := m.get(id)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		view, ok, err := m.cancel(id)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "job not found")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	writeJSONError(w, http.StatusNotFound, "unknown job route")
}
