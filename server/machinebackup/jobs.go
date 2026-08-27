package machinebackup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	JobQueued   = "queued"
	JobRunning  = "running"
	JobDone     = "done"
	JobError    = "error"
	JobCanceled = "canceled"

	PhaseWalk  = "walk"
	PhasePack  = "pack"
	PhaseReady = "ready"

	defaultJobHeartbeat = 2 * time.Second
	defaultJobStale     = 30 * time.Minute
)

// BackupJobView is the JSON snapshot for poll clients.
type BackupJobView struct {
	ID           string `json:"job_id"`
	Status       string `json:"status"`
	Phase        string `json:"phase,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	FilesTotal   int    `json:"files_total,omitempty"`
	FilesDone    int    `json:"files_done,omitempty"`
	BytesDone    int64  `json:"bytes_done,omitempty"`
	ArchiveBytes int64  `json:"archive_bytes,omitempty"`
	ArchiveToken string `json:"archive_token,omitempty"`
	Error        string `json:"error,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
	Archive      bool   `json:"archive,omitempty"`
}

// BackupJobList is GET /backup/jobs.
type BackupJobList struct {
	Active *BackupJobView `json:"active"`
	Last   *BackupJobView `json:"last"`
}

type backupJob struct {
	id           string
	req          BackupStreamRequest
	status       string
	phase        string
	startedAt    time.Time
	updatedAt    time.Time
	finishedAt   time.Time
	progressAt   time.Time
	filesTotal   int
	filesDone    int
	bytesDone    int64
	archiveBytes int64
	archiveToken string
	archivePath  string
	errMsg       string
	stopStatus   string
	stopErr      string
	ctx          context.Context
	cancel       context.CancelFunc
}

func (j *backupJob) view() BackupJobView {
	return BackupJobView{
		ID:           j.id,
		Status:       j.status,
		Phase:        j.phase,
		StartedAt:    rfc3339(j.startedAt),
		UpdatedAt:    rfc3339(j.updatedAt),
		FinishedAt:   rfc3339(j.finishedAt),
		FilesTotal:   j.filesTotal,
		FilesDone:    j.filesDone,
		BytesDone:    j.bytesDone,
		ArchiveBytes: j.archiveBytes,
		ArchiveToken: j.archiveToken,
		Error:        j.errMsg,
		DryRun:       j.req.DryRun || !j.req.Archive,
		Archive:      j.req.Archive,
	}
}

type jobManager struct {
	homeFn         func() string
	heartbeatEvery time.Duration
	staleAfter     time.Duration
	now            func() time.Time
	beforeWalk     func(*backupJob)

	mu     sync.Mutex
	active *backupJob
	last   *backupJob
}

func newJobManager(homeFn func() string) *jobManager {
	return &jobManager{
		homeFn:         homeFn,
		heartbeatEvery: defaultJobHeartbeat,
		staleAfter:     defaultJobStale,
		now:            time.Now,
	}
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func newJobID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// start begins a backup job. If one is already queued/running, started is false
// and the existing job is returned (caller should HTTP 409).
func (m *jobManager) start(req BackupStreamRequest) (view BackupJobView, started bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && (m.active.status == JobQueued || m.active.status == JobRunning) {
		return m.active.view(), false
	}
	now := m.now()
	ctx, cancel := context.WithCancel(context.Background())
	job := &backupJob{
		id:         newJobID(),
		req:        req,
		status:     JobQueued,
		startedAt:  now,
		updatedAt:  now,
		progressAt: now,
		ctx:        ctx,
		cancel:     cancel,
	}
	m.active = job
	go m.run(job)
	return job.view(), true
}

func (m *jobManager) get(id string) (BackupJobView, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && m.active.id == id {
		return m.active.view(), true
	}
	if m.last != nil && m.last.id == id {
		return m.last.view(), true
	}
	return BackupJobView{}, false
}

func (m *jobManager) list() BackupJobList {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out BackupJobList
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

func (m *jobManager) cancel(id string) (BackupJobView, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.findLocked(id)
	if job == nil {
		return BackupJobView{}, false, nil
	}
	if job.status != JobQueued && job.status != JobRunning {
		return job.view(), true, fmt.Errorf("job is %s", job.status)
	}
	m.requestStopLocked(job, JobCanceled, "canceled")
	return job.view(), true, nil
}

func (m *jobManager) findLocked(id string) *backupJob {
	if m.active != nil && m.active.id == id {
		return m.active
	}
	if m.last != nil && m.last.id == id {
		return m.last
	}
	return nil
}

func (m *jobManager) requestStopLocked(job *backupJob, status, msg string) {
	if job.stopStatus != "" {
		return
	}
	job.stopStatus = status
	job.stopErr = msg
	if job.cancel != nil {
		job.cancel()
	}
}

func (m *jobManager) isStopped(job *backupJob) (bool, string, string) {
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

func (m *jobManager) run(job *backupJob) {
	defer job.cancel()
	m.setRunning(job, PhaseWalk)
	m.startHeartbeat(job)

	if m.beforeWalk != nil {
		m.beforeWalk(job)
	}
	if stopped, st, msg := m.isStopped(job); stopped {
		m.finish(job, st, msg)
		return
	}

	home := strings.TrimSpace(m.homeFn())
	gitOpts := GitScanOptions{
		SkipGitDirsScan:     job.req.SkipGitDirsScan,
		GitDirsScanMaxDepth: job.req.GitDirsScanMaxDepth,
	}
	prepared, err := prepareBackup(home, job.req.Exclude, job.req.Include, gitOpts)
	if err != nil {
		m.finish(job, JobError, err.Error())
		return
	}
	m.setWalkTotals(job, prepared)
	if stopped, st, msg := m.isStopped(job); stopped {
		m.finish(job, st, msg)
		return
	}

	if job.req.DryRun || !job.req.Archive {
		m.setPhase(job, PhaseReady)
		m.finish(job, JobDone, "")
		return
	}

	m.setPhase(job, PhasePack)
	path, n, err := packArchiveFile(prepared, func(_, _ string) error {
		if stopped, _, _ := m.isStopped(job); stopped {
			return context.Canceled
		}
		m.notePackProgress(job)
		return nil
	})
	if err != nil {
		if stopped, st, msg := m.isStopped(job); stopped {
			m.finish(job, st, msg)
			return
		}
		if errors.Is(err, context.Canceled) {
			m.finish(job, JobCanceled, "canceled")
			return
		}
		m.finish(job, JobError, err.Error())
		return
	}
	token, err := registerArchiveSession(path)
	if err != nil {
		os.Remove(path)
		m.finish(job, JobError, err.Error())
		return
	}
	m.setArchive(job, token, path, n)
	m.setPhase(job, PhaseReady)
	m.finish(job, JobDone, "")
}

func (m *jobManager) startHeartbeat(job *backupJob) {
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
					m.checkStaleLocked(job)
				}
				m.mu.Unlock()
			}
		}
	}()
}

func (m *jobManager) checkStaleLocked(job *backupJob) {
	if m.staleAfter <= 0 {
		return
	}
	if job.status != JobRunning {
		return
	}
	if m.now().Sub(job.progressAt) <= m.staleAfter {
		return
	}
	m.requestStopLocked(job, JobError, "backup job stale: no progress")
}

func (m *jobManager) setRunning(job *backupJob, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	job.status = JobRunning
	job.phase = phase
	job.updatedAt = now
	job.progressAt = now
}

func (m *jobManager) setPhase(job *backupJob, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	job.phase = phase
	job.updatedAt = now
	job.progressAt = now
}

func (m *jobManager) setWalkTotals(job *backupJob, prepared *backupPrepared) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if prepared != nil && prepared.Plan != nil {
		job.filesTotal = prepared.Plan.GrandTotal.Files
		job.bytesDone = 0
	}
	job.updatedAt = now
	job.progressAt = now
}

func (m *jobManager) notePackProgress(job *backupJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	job.filesDone++
	job.updatedAt = now
	job.progressAt = now
}

func (m *jobManager) setArchive(job *backupJob, token, path string, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job.archiveToken = token
	job.archivePath = path
	job.archiveBytes = n
	job.updatedAt = m.now()
	job.progressAt = job.updatedAt
}

func (m *jobManager) finish(job *backupJob, status, errMsg string) {
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
	if status != JobDone && job.archivePath != "" {
		os.Remove(job.archivePath)
		job.archivePath = ""
		job.archiveToken = ""
	}
	if m.active != nil && m.active.id == job.id {
		m.last = job
		m.active = nil
	}
}

func registerBackupJobRoutes(mux *http.ServeMux, m *jobManager) {
	mux.HandleFunc("/api/remote-agent/machine/backup/jobs", func(w http.ResponseWriter, r *http.Request) {
		m.handleCollection(w, r)
	})
	mux.HandleFunc("/api/remote-agent/machine/backup/jobs/", func(w http.ResponseWriter, r *http.Request) {
		m.handleItem(w, r)
	})
}

func (m *jobManager) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, m.list())
	case http.MethodPost:
		var req BackupStreamRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
				writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
				return
			}
		}
		if req.Exclude == nil {
			req.Exclude = []string{}
		}
		if req.Include == nil {
			req.Include = []string{}
		}
		view, started := m.start(req)
		if !started {
			body := view
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  "backup job already running",
				"job_id": body.ID,
				"job":    body,
			})
			return
		}
		writeJSON(w, http.StatusAccepted, view)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (m *jobManager) handleItem(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/remote-agent/machine/backup/jobs/")
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
