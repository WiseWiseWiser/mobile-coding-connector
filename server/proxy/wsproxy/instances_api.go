package wsproxy

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/xhd2015/agent-pro/agent/streaming/sse"
)

// RegisterInstanceAPI mounts the multi-instance CRUD + control endpoints.
func RegisterInstanceAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ws-proxy/instances", handleListInstances)
	mux.HandleFunc("POST /api/ws-proxy/instances", handleCreateInstance)
	mux.HandleFunc("PUT /api/ws-proxy/instances/{id}", handleUpdateInstance)
	mux.HandleFunc("DELETE /api/ws-proxy/instances/{id}", handleDeleteInstance)
	mux.HandleFunc("POST /api/ws-proxy/instances/{id}/start", handleStartInstance)
	mux.HandleFunc("POST /api/ws-proxy/instances/{id}/start/stream", handleStartInstanceStream)
	mux.HandleFunc("POST /api/ws-proxy/instances/{id}/stop", handleStopInstance)
	mux.HandleFunc("POST /api/ws-proxy/instances/{id}/stop/stream", handleStopInstanceStream)
	mux.HandleFunc("GET /api/ws-proxy/instances/{id}/qrcode", handleInstanceQRCode)
	mux.HandleFunc("GET /api/ws-proxy/instances/{id}/vmess-link", handleInstanceVMessLink)
}

// handleStartInstanceStream runs the start with live SSE progress lines:
// ensure binary → xray → ensure tunnel → DNS/ingress → done. Status events
// drive the UI step bar (onCustom in the frontend consumer).
func handleStartInstanceStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sw := sse.NewWriter(w)
	if sw == nil {
		writeInstanceErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	logFn := func(msg string) { sw.SendLog(msg) }
	statusFn := func(step string) { sw.SendStatus(step, nil) }

	instances, err := LoadInstances()
	if err != nil {
		sw.SendError(err.Error())
		sw.SendDone(map[string]string{"status": "error"})
		return
	}
	inst, _ := FindInstance(instances, id)
	if inst == nil {
		sw.SendError("instance not found: " + id)
		sw.SendDone(map[string]string{"status": "error"})
		return
	}

	statusFn("binary")
	if err := GetManager().StartInstanceWithLog(id, func(msg string) {
		logFn(msg)
		switch {
		case strings.Contains(msg, "ensure xray binary"):
			statusFn("binary")
		case strings.Contains(msg, "xray up"):
			statusFn("xray")
		case strings.Contains(msg, "ensure extension tunnel"), strings.Contains(msg, "DNS route"):
			statusFn("tunnel")
		case strings.Contains(msg, "ingress"), strings.Contains(msg, "published"):
			statusFn("publish")
		}
	}); err != nil {
		sw.SendError(err.Error())
		hint := instanceStartHint(err)
		if hint != "" {
			sw.SendLog("hint: " + hint)
		}
		sw.SendDone(map[string]string{"status": "error"})
		return
	}

	instances, _ = LoadInstances()
	fresh, _ := FindInstance(instances, id)
	publicURL := ""
	port := 0
	if fresh != nil {
		publicURL = fresh.PublicURL
		port = fresh.ListenPort
	}
	statusFn("done")
	sw.SendDone(map[string]string{
		"status":     "running",
		"public_url": publicURL,
		"port":       strconv.Itoa(port),
	})
}

// handleStopInstanceStream stops the instance with a short streamed tail.
func handleStopInstanceStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sw := sse.NewWriter(w)
	if sw == nil {
		writeInstanceErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	sw.SendLog("stopping instance " + id + "…")
	if err := GetManager().StopInstance(id); err != nil {
		sw.SendError(err.Error())
		sw.SendDone(map[string]string{"status": "error"})
		return
	}
	sw.SendLog("stopped")
	sw.SendDone(map[string]string{"status": "stopped"})
}

// instanceStartHint turns common start failures into actionable guidance.
func instanceStartHint(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "backend 不可达"), strings.Contains(msg, "i/o timeout"):
		return "backend unreachable — check the target HTTP proxy is running and the port is reachable"
	case strings.Contains(msg, "extension tunnel"):
		return "cloudflare tunnel setup failed — check cert.pem (~/.cloudflared) and cloudflared availability"
	case strings.Contains(msg, "DNS route"):
		return "DNS route failed — verify the zone credentials in ~/.cloudflared cover this domain"
	default:
		return ""
	}
}

func writeInstanceErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": msg})
}

func handleListInstances(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]interface{}{
		"instances": GetManager().ListInstancesWithStatus(),
	})
}

func handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		BackendType string `json:"backend_type"`
		BackendURL  string `json:"backend_url"`
		Subdomain   string `json:"subdomain"`
		AutoStart   bool   `json:"auto_start"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInstanceErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	instances, err := LoadInstances()
	if err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := Instance{
		Name:        req.Name,
		BackendType: BackendType(req.BackendType),
		BackendURL:  req.BackendURL,
		Subdomain:   req.Subdomain,
		AutoStart:   req.AutoStart,
		Enabled:     true,
	}
	if err := ValidateInstance(&inst); err != nil {
		writeInstanceErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if existing, _ := FindInstance(instances, inst.ID); existing != nil {
		writeInstanceErr(w, http.StatusConflict, "instance id already exists: "+inst.ID)
		return
	}
	// Subdomain uniqueness across instances.
	for _, other := range instances {
		if other.Subdomain == inst.Subdomain {
			writeInstanceErr(w, http.StatusConflict, "subdomain already used by instance "+other.ID)
			return
		}
	}
	instances = append(instances, inst)
	if err := SaveInstances(instances); err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inst.AutoStart {
		_ = GetManager().StartInstance(inst.ID)
	}
	writeJSON(w, map[string]interface{}{"instance": inst})
}

func loadInstanceFromPath(r *http.Request) (*Instance, []Instance, error) {
	id := r.PathValue("id")
	instances, err := LoadInstances()
	if err != nil {
		return nil, nil, err
	}
	inst, _ := FindInstance(instances, id)
	if inst == nil {
		return nil, nil, errNotFoundInstance(id)
	}
	return inst, instances, nil
}

func errNotFoundInstance(id string) error {
	return &simpleError{"instance not found: " + id}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	inst, instances, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		Name        *string `json:"name"`
		BackendType *string `json:"backend_type"`
		BackendURL  *string `json:"backend_url"`
		Subdomain   *string `json:"subdomain"`
		AutoStart   *bool   `json:"auto_start"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInstanceErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	wasRunning := inst.PublicURL != ""
	if req.Name != nil {
		inst.Name = *req.Name
	}
	if req.BackendType != nil {
		inst.BackendType = BackendType(*req.BackendType)
	}
	if req.BackendURL != nil {
		inst.BackendURL = *req.BackendURL
	}
	if req.Subdomain != nil && *req.Subdomain != inst.Subdomain {
		for _, other := range instances {
			if other.ID != inst.ID && other.Subdomain == strings.ToLower(strings.TrimSpace(*req.Subdomain)) {
				writeInstanceErr(w, http.StatusConflict, "subdomain already used by instance "+other.ID)
				return
			}
		}
		inst.Subdomain = *req.Subdomain
		// A changed subdomain invalidates the pinned hostname.
		inst.Hostname = ""
		inst.PublicURL = ""
	}
	if req.AutoStart != nil {
		inst.AutoStart = *req.AutoStart
	}
	if req.Enabled != nil {
		inst.Enabled = *req.Enabled
	}
	if err := ValidateInstance(inst); err != nil {
		writeInstanceErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := SaveInstances(instances); err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Backend/subdomain changes require a restart of the instance.
	if wasRunning && req.BackendType != nil || wasRunning && req.BackendURL != nil || wasRunning && req.Subdomain != nil {
		_ = GetManager().StartInstance(inst.ID)
	}
	writeJSON(w, map[string]interface{}{"instance": *inst})
}

func handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	inst, instances, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	_ = GetManager().StopInstance(inst.ID)
	idx := -1
	for i := range instances {
		if instances[i].ID == inst.ID {
			idx = i
			break
		}
	}
	instances = append(instances[:idx], instances[idx+1:]...)
	if err := SaveInstances(instances); err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "deleted", "id": inst.ID})
}

func handleStartInstance(w http.ResponseWriter, r *http.Request) {
	inst, _, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	if err := GetManager().StartInstance(inst.ID); err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	instances, _ := LoadInstances()
	fresh, _ := FindInstance(instances, inst.ID)
	writeJSON(w, map[string]interface{}{"instance": fresh})
}

func handleStopInstance(w http.ResponseWriter, r *http.Request) {
	inst, _, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	if err := GetManager().StopInstance(inst.ID); err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	instances, _ := LoadInstances()
	fresh, _ := FindInstance(instances, inst.ID)
	writeJSON(w, map[string]interface{}{"instance": fresh})
}

func handleInstanceQRCode(w http.ResponseWriter, r *http.Request) {
	inst, _, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	link, _, err := GetManager().InstanceVMessLink(inst.ID)
	if err != nil {
		writeInstanceErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	png, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		writeInstanceErr(w, http.StatusInternalServerError, "failed to encode QR: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func handleInstanceVMessLink(w http.ResponseWriter, r *http.Request) {
	inst, _, err := loadInstanceFromPath(r)
	if err != nil {
		writeInstanceErr(w, http.StatusNotFound, err.Error())
		return
	}
	link, cfg, err := GetManager().InstanceVMessLink(inst.ID)
	if err != nil {
		writeInstanceErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{
		"vmess_link": link,
		"host":       cfg.Host,
		"uuid":       cfg.UUID,
		"path":       cfg.Path,
		"tls":        cfg.TLS,
	})
}
