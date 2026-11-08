package diskmon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
	diskhuman "github.com/xhd2015/dot-pkgs/go-pkgs/disk/human"
	"github.com/xhd2015/dot-pkgs/go-pkgs/disk/reclaim"
)

// defaultCleanup is the process-wide cleanup service behind the report panel.
var defaultCleanup = newCleanupService()

func registerCleanupAPI(mux *http.ServeMux) {
	(&cleanupAPI{service: defaultCleanup}).register(mux)
}

type cleanupAPI struct {
	service *cleanupService
}

func (a *cleanupAPI) register(mux *http.ServeMux) {
	mux.HandleFunc("/api/server/disk/cleanup/plan", a.handlePlan)
	mux.HandleFunc("/api/server/disk/cleanup/measure", a.handleMeasure)
	mux.HandleFunc("/api/server/disk/cleanup/run", a.handleRun)
}

// handlePlan returns the measured cleanup classes. ?refresh=1 re-measures.
func (a *cleanupAPI) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	force := r.URL.Query().Get("refresh") == "1"
	plan, err := a.service.Plan(force)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, plan)
}

// handleMeasure sizes a subset (typically the system classes, which are too slow
// to include in the initial plan) and caches the result for the next /plan.
func (a *cleanupAPI) handleMeasure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := decodeCleanupBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	err := a.service.measureClass(req.IDs)
	if errors.Is(err, errCleanupMeasureTimeout) {
		// The probe is still running; its numbers land in the cache when it
		// finishes, so the page can poll /plan instead of holding the request.
		writeJSON(w, http.StatusOK, map[string]any{"pending": true})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"pending": false})
}

// handleRun clears the selected classes and streams one progress frame per
// step. Sudo="never" runs unprivileged only: admin classes come back in
// skipped[] with their command so the user can still clear them by hand.
func (a *cleanupAPI) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		IDs  []string `json:"ids"`
		Sudo string   `json:"sudo"`
	}
	if err := decodeCleanupBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	ids, skipped, err := a.resolveRun(req.IDs, req.Sudo)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no clearable classes selected"})
		return
	}
	if !a.service.tryStart() {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "a cleanup is already running"})
		return
	}
	defer a.service.finish()

	out := progress.NewWriter(w)
	if out == nil {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	labels, err := a.service.labels()
	if err != nil {
		_ = out.EmitError(err.Error())
		return
	}

	_ = out.EmitMeta(map[string]any{
		"home":        homeFor(a.service),
		"steps":       len(ids),
		"sudo":        req.Sudo,
		"sudoGranted": req.Sudo == cleanupSudoAsk && sudoAlreadyGranted(),
	})
	_ = out.EmitSection("Clearing")

	result, runErr := a.service.run(reclaim.Options{
		Only:          ids,
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		ElevateMode:   reclaim.ElevateOSAScript,
		ElevatePrompt: "ai-critic",
		OnStep:        cleanupStepEmitter(out, labels),
	})

	// The engine reports per-step failures inside Result; only a failure to run
	// at all (bad selection, unusable temp dir) is an error frame.
	if runErr != nil {
		_ = out.EmitError(runErr.Error())
		return
	}
	for _, skip := range skipped {
		_ = out.EmitProgress(progress.Item{
			ID:     skip.ID,
			Layer:  "cleanup",
			Name:   skip.Label,
			Status: "skipped",
			Detail: skip.Reason,
			Hint:   skip.Command,
		})
	}
	done := CleanupRunResult{Skipped: skipped}
	for _, st := range result.Steps {
		item := CleanupStepResult{ID: st.ID, Label: labels[st.ID], Freed: st.Freed, Error: st.Error}
		done.Steps = append(done.Steps, item)
		done.FreedBytes += st.Freed
	}
	if done.FreedBytes > 0 {
		_ = out.EmitLog(fmt.Sprintf("Freed %s", humanCleanupBytes(done.FreedBytes)), true)
	}
	_ = out.EmitDone(cleanupDonePayload(done))
}

// resolveRun validates the requested ids and splits off the classes the sudo
// policy forbids. Read-only inventory rows are always rejected: they must never
// reach the engine.
func (a *cleanupAPI) resolveRun(reqIDs []string, sudo string) ([]string, []CleanupSkip, error) {
	if sudo == "" {
		sudo = cleanupSudoAsk
	}
	if sudo != cleanupSudoAsk && sudo != cleanupSudoNever {
		return nil, nil, fmt.Errorf("unknown sudo policy %q", sudo)
	}
	infos, err := a.service.stepInfos()
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]reclaim.StepInfo, len(infos))
	for _, in := range infos {
		byID[in.ID] = in
	}
	var ids []string
	var skipped []CleanupSkip
	seen := map[string]bool{}
	for _, id := range reqIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		info, ok := byID[id]
		if !ok {
			return nil, nil, fmt.Errorf("unknown cleanup class %q", id)
		}
		if info.ReadOnly {
			return nil, nil, fmt.Errorf("cleanup class %q is read-only", id)
		}
		if sudo == cleanupSudoNever && info.NeedsAdmin {
			skipped = append(skipped, CleanupSkip{
				ID:      id,
				Label:   cleanupLabel(info),
				Reason:  "needs administrator authorization",
				Command: cleanupHeadline(homeFor(a.service), info),
			})
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 && len(skipped) == 0 {
		return nil, nil, fmt.Errorf("no cleanup class ids")
	}
	return ids, skipped, nil
}

// cleanupStepEmitter turns engine step events into progress frames.
func cleanupStepEmitter(out *progress.Writer, labels map[string]string) func(reclaim.StepEvent) {
	return func(ev reclaim.StepEvent) {
		name := labels[ev.ID]
		if name == "" {
			name = ev.ID
		}
		switch ev.Type {
		case "start":
			_ = out.EmitProgress(progress.Item{ID: ev.ID, Layer: "cleanup", Name: name, Status: "running"})
		case "done":
			status := "done"
			detail := ""
			if ev.Error != "" {
				status = "warn"
				detail = ev.Error
			}
			if ev.Freed > 0 {
				detail = "freed " + humanCleanupBytes(ev.Freed)
				if ev.Error != "" {
					detail += " · " + ev.Error
				}
			}
			_ = out.EmitProgress(progress.Item{ID: ev.ID, Layer: "cleanup", Name: name, Status: status, Detail: detail})
		}
	}
}

// cleanupDonePayload flattens the run result into the done frame.
func cleanupDonePayload(done CleanupRunResult) map[string]any {
	return map[string]any{
		"freedBytes": done.FreedBytes,
		"freed":      humanCleanupBytes(done.FreedBytes),
		"steps":      done.Steps,
		"skipped":    done.Skipped,
	}
}

func decodeCleanupBody(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBody = 1 << 16
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func (s *cleanupService) stepInfos() ([]reclaim.StepInfo, error) {
	home, err := s.home()
	if err != nil {
		return nil, err
	}
	return reclaim.StepInfos(home), nil
}

func (s *cleanupService) labels() (map[string]string, error) {
	infos, err := s.stepInfos()
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(infos))
	for _, in := range infos {
		labels[in.ID] = cleanupLabel(in)
	}
	return labels, nil
}

func homeFor(s *cleanupService) string {
	home, err := s.home()
	if err != nil {
		return ""
	}
	return home
}

func humanCleanupBytes(n int64) string { return diskhuman.Size(n) }
