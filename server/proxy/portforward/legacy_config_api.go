package portforward

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/xhd2015/ai-critic/server/config"
)

// The old names/auth endpoints remain adapters over the canonical v2 store.
// Numeric aliases are exported only for the default local mapping; network
// mappings always use IDs so two origins never share saved credentials.
func handleMappingLegacyConfig(w http.ResponseWriter, r *http.Request, auth bool) {
	mappingMutationMu.Lock()
	defer mappingMutationMu.Unlock()
	store, err := config.LoadMappings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	selector := r.URL.Query().Get("id")
	if selector == "" {
		selector = r.URL.Query().Get("port")
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet && selector == "" {
		if auth {
			out := map[string]config.PortAuth{}
			for id, rec := range store.Mappings {
				if !rec.Auth.Required {
					continue
				}
				out[id] = rec.Auth
				if id == config.LocalMappingID(rec.Target.Port) {
					out[strconv.Itoa(rec.Target.Port)] = rec.Auth
				}
			}
			_ = json.NewEncoder(w).Encode(out)
		} else {
			out := map[string]string{}
			for id, rec := range store.Mappings {
				name := savedMappingName(rec)
				if name == "" {
					continue
				}
				out[id] = name
				if id == config.LocalMappingID(rec.Target.Port) {
					out[strconv.Itoa(rec.Target.Port)] = name
				}
			}
			_ = json.NewEncoder(w).Encode(out)
		}
		return
	}
	var name string
	if r.Method == http.MethodPost && !auth {
		var req struct {
			Port   int    `json:"port"`
			ID     string `json:"id"`
			Domain string `json:"domain"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Domain == "" {
			http.Error(w, "port/id and domain are required", 400)
			return
		}
		selector = req.ID
		if selector == "" {
			selector = strconv.Itoa(req.Port)
		}
		name = req.Domain
	}
	rec, err := config.ResolveMapping(store, selector)
	if err != nil {
		port, e := strconv.Atoi(selector)
		if e != nil || port < 1 || port > 65535 {
			http.Error(w, err.Error(), 409)
			return
		}
		// Do not swallow ambiguity as a new localhost entry.
		for _, candidate := range store.Mappings {
			if candidate.Target.Port == port {
				http.Error(w, err.Error(), 409)
				return
			}
		}
		rec = config.MappingRecord{ID: config.LocalMappingID(port), Target: config.OriginTarget{Host: "127.0.0.1", Scheme: "http", Port: port}}
	}
	if r.Method == http.MethodGet {
		if auth {
			_ = json.NewEncoder(w).Encode(map[string]any{"port": selector, "id": rec.ID, "auth": rec.Auth})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{"port": selector, "id": rec.ID, "domain": savedMappingName(rec)})
		}
		return
	}
	if r.Method == http.MethodDelete {
		if auth {
			rec.Auth = config.PortAuth{}
		} else {
			empty := ""
			rec.SavedName = &empty
		}
	} else if r.Method == http.MethodPost && !auth {
		rec.SavedName = &name
		if !rec.Desired {
			rec.Hostname = name
		}
	} else {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if err = config.UpdateMappings(func(s *config.MappingStore) error { s.Mappings[rec.ID] = rec; return nil }); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "port": rec.Target.Port, "id": rec.ID, "domain": savedMappingName(rec)})
}

func savedMappingName(rec config.MappingRecord) string {
	if rec.SavedName != nil {
		return *rec.SavedName
	}
	return rec.Hostname
}
