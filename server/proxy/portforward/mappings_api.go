package portforward

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/domains"
	"github.com/xhd2015/ai-critic/server/proxy/origin"
)

// Serialize control-plane updates with restore so an update cannot resurrect
// an old target or leave half of a delete visible to the watchdog.
var mappingMutationMu sync.Mutex

type targetCheck struct {
	Target       config.OriginTarget `json:"target"`
	TCPReachable bool                `json:"tcpReachable"`
	Error        string              `json:"error,omitempty"`
}

func checkTarget(ctx context.Context, t config.OriginTarget) targetCheck {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out := targetCheck{Target: t}
	err := origin.Probe(ctx, t)
	out.TCPReachable = err == nil
	if err != nil {
		out.Error = err.Error()
	}
	return out
}
func handleTargetCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct {
		Target config.OriginTarget `json:"target"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	target, err := req.Target.Normalize(0)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(checkTarget(r.Context(), target))
}

func handleMappingWrite(w http.ResponseWriter, r *http.Request) {
	var req addPortRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32768)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", 400)
		return
	}
	mappingMutationMu.Lock()
	defer mappingMutationMu.Unlock()
	store, err := config.LoadMappings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var previous *config.MappingRecord
	previousRunning := false
	if r.Method == http.MethodPut {
		selector := r.URL.Query().Get("id")
		if selector == "" {
			selector = r.URL.Query().Get("port")
		}
		record, e := config.ResolveMapping(store, selector)
		if e != nil {
			http.Error(w, e.Error(), 409)
			return
		}
		previous = &record
		defaultManager.mu.Lock()
		previousRunning = defaultManager.tunnels[record.ID] != nil
		defaultManager.mu.Unlock()
	}
	rec := config.MappingRecord{Provider: req.Provider, Hostname: req.Label, Desired: true}
	if previous != nil {
		rec = *previous
		rec.Desired = true
		if req.Provider != "" {
			rec.Provider = req.Provider
		}
		if req.Label != "" {
			rec.Hostname = req.Label
		}
	}
	if req.Target != nil {
		if previous == nil {
			rec.Target = *req.Target
		} else {
			if req.Target.Host != "" {
				rec.Target.Host = req.Target.Host
			}
			if req.Target.Scheme != "" {
				rec.Target.Scheme = req.Target.Scheme
			}
			if req.Target.Port != 0 {
				rec.Target.Port = req.Target.Port
			}
		}
	} else if previous == nil {
		rec.Target = config.OriginTarget{Port: req.Port}
	} else if req.Port != 0 {
		rec.Target.Port = req.Port
	}
	target, e := rec.Target.Normalize(req.Port)
	if e != nil {
		http.Error(w, e.Error(), 400)
		return
	}
	rec.Target = target
	if rec.Provider == "" {
		rec.Provider = ProviderLocaltunnel
	}
	if req.Subdomain != "" && (rec.Provider == ProviderCloudflareOwned || rec.Provider == ProviderCloudflareTunnel) {
		base := req.BaseDomain
		if base == "" {
			owned := cloudflare.GetOwnedDomains()
			if len(owned) > 0 {
				base = owned[0]
			}
		}
		if base != "" {
			rec.Hostname = req.Subdomain + "." + base
		}
	}
	if rec.Hostname == "" {
		rec.Hostname = fmt.Sprintf("Port %d", rec.Target.Port)
	}
	if strings.ContainsAny(rec.Hostname, "\r\n\t/") {
		http.Error(w, "invalid hostname or label", 400)
		return
	}
	if previous == nil {
		if target.Host == "127.0.0.1" && target.Scheme == "http" {
			rec.ID = config.LocalMappingID(target.Port)
		} else {
			rec.ID, err = config.NewMappingID()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		if old, ok := store.Mappings[rec.ID]; ok && !old.Desired {
			rec.Auth = old.Auth
		}
	}
	if req.Auth != nil {
		rec.Auth, err = resolveMappingRecordAuth(rec.Auth, *req.Auth)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}
	if previous != nil {
		_ = defaultManager.RemoveSelector(previous.ID)
	}
	savedName := rec.Hostname
	rec.SavedName = &savedName
	forward := recordForward(rec)
	pf, err := defaultManager.AddForward(forward)
	if err != nil {
		if previous != nil && previousRunning {
			_, _ = defaultManager.AddForward(recordForward(*previous))
		}
		http.Error(w, err.Error(), 409)
		return
	}
	err = config.UpdateMappings(func(s *config.MappingStore) error {
		for id, old := range s.Mappings {
			if id != rec.ID && isCloudflareHostnameProvider(old.Provider) && isCloudflareHostnameProvider(rec.Provider) && sameForwardHostname(old.Hostname, rec.Hostname) {
				old.Desired = false
				s.Mappings[id] = old
			}
		}
		s.Mappings[rec.ID] = rec
		return nil
	})
	if err != nil {
		_ = defaultManager.RemoveSelector(rec.ID)
		if previous != nil && previousRunning {
			_, _ = defaultManager.AddForward(recordForward(*previous))
		}
		http.Error(w, err.Error(), 500)
		return
	}
	// The old files are migration inputs only. Do not persist network targets or
	// credentials by port: another host may intentionally use the same port.
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}
	out := struct {
		*PortForward
		Origin targetCheck `json:"origin"`
	}{pf, checkTarget(r.Context(), target)}
	_ = json.NewEncoder(w).Encode(out)
}

func recordForward(rec config.MappingRecord) ForwardRequest {
	req := ForwardRequest{ID: rec.ID, Port: rec.Target.Port, Target: &rec.Target, Label: rec.Hostname, Provider: rec.Provider, Auth: rec.Auth}
	// A plain publish of this management server must keep its existing direct
	// loopback path. Treating it as a different origin strips the very bearer
	// token/cookie that the management endpoint needs. Never apply this exception
	// to a network target, another local application, HTTPS, or an auth gate.
	if rec.Target.Host == "127.0.0.1" && rec.Target.Scheme == "http" &&
		rec.Target.Port == domains.GetServerPort() && !rec.Auth.Required {
		req.Target = nil
	}
	return req
}
func resolveMappingRecordAuth(previous, req config.PortAuth) (config.PortAuth, error) {
	if !req.Required {
		return config.PortAuth{}, nil
	}
	if req.Token == "" && req.TokenMode != "shared" && req.TokenMode != "custom" && previous.Required && previous.Token != "" {
		if strings.TrimSpace(req.User) != "" {
			previous.User = strings.TrimSpace(req.User)
		}
		return previous, nil
	}
	if req.TokenMode == config.PortAuthShared {
		req.Token = ""
		return req, nil
	}
	if req.Token == "" && req.TokenMode == config.PortAuthCustom {
		return req, fmt.Errorf("custom auth token must be non-empty")
	}
	if req.Token == "" {
		token, err := config.GenerateEphemeralToken()
		if err != nil {
			return req, err
		}
		req.Token = token
		req.TokenMode = config.PortAuthEphemeral
		req.Ephemeral = true
	} else if req.TokenMode == "" {
		req.TokenMode = config.PortAuthCustom
	}
	return req, nil
}

func handleMappingRemove(w http.ResponseWriter, r *http.Request) {
	selector := r.URL.Query().Get("id")
	if selector == "" {
		selector = r.URL.Query().Get("port")
	}
	if selector == "" {
		var body struct {
			Port int    `json:"port"`
			ID   string `json:"id"`
		}
		if json.NewDecoder(r.Body).Decode(&body) == nil {
			selector = body.ID
			if selector == "" && body.Port > 0 {
				selector = strconv.Itoa(body.Port)
			}
		}
	}
	if selector == "" {
		http.Error(w, "id or port parameter is required", 400)
		return
	}
	mappingMutationMu.Lock()
	defer mappingMutationMu.Unlock()
	store, err := config.LoadMappings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	rec, err := config.ResolveMapping(store, selector)
	if err != nil {
		// Legacy service callers may have created a runtime-only forward.
		if e := defaultManager.RemoveSelector(selector); e != nil {
			http.Error(w, err.Error(), 409)
			return
		}
	} else {
		err = config.UpdateMappings(func(s *config.MappingStore) error {
			if r.URL.Query().Get("forget") == "true" {
				delete(s.Mappings, rec.ID)
			} else {
				rec.Desired = false
				s.Mappings[rec.ID] = rec
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = defaultManager.RemoveSelector(rec.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}

// ReadSavedMappings exposes inactive mappings without leaking saved tokens.
func handleSavedMappings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}
	store, err := config.LoadMappings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for id, rec := range store.Mappings {
		rec.Auth.Token = ""
		store.Mappings[id] = rec
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(store)
}
