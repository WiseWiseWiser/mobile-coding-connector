package usage

import (
	"encoding/json"
	"net/http"

	"github.com/xhd2015/ai-critic/macosapp/codexusage"
	"github.com/xhd2015/ai-critic/macosapp/debuglog"
	"github.com/xhd2015/ai-critic/macosapp/grokusage"
	"github.com/xhd2015/ai-critic/macosapp/usageitems"
	"github.com/xhd2015/ai-critic/server/config"
)

var (
	grokService  = grokusage.NewService()
	codexService = codexusage.NewService()
	itemsService = usageitems.NewService(usageitems.NewStore(config.UsageItemsFile))
)

type debugSettingsResponse struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type debugSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

// RegisterAPI registers grok/codex usage, usage items, and debug log settings.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/grok/usage", handleGrokUsage)
	mux.HandleFunc("/api/codex/usage", handleCodexUsage)
	mux.HandleFunc("/api/usage/items", handleUsageItems)
	mux.HandleFunc("/api/usage/items/add", handleUsageItemsAdd)
	mux.HandleFunc("/api/usage/items/update", handleUsageItemsUpdate)
	mux.HandleFunc("/api/usage/items/remove", handleUsageItemsRemove)
	mux.HandleFunc("/api/usage/items/default", handleUsageItemsDefault)
	mux.HandleFunc("/api/debug/log", handleDebugLog)
}

// Start begins background refresh loops for usage services.
func Start() {
	grokService.Start()
	codexService.Start()
	itemsService.Start()
}

// Stop ends background refresh loops.
func Stop() {
	grokService.Stop()
	codexService.Stop()
	itemsService.Stop()
}

func handleGrokUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	grokService.EnsureFetch()
	resp := grokService.Get()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleCodexUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	codexService.EnsureFetch()
	resp := codexService.Get()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleUsageItems serves GET /api/usage/items: every registry item with the
// menu-bar title and dropdown text the app renders verbatim.
func handleUsageItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := itemsService.List()
	if err != nil {
		writeUsageError(w, http.StatusInternalServerError, err)
		return
	}
	writeUsageJSON(w, http.StatusOK, list)
}

func handleUsageItemsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req usageItemRequest
	if err := decodeUsageJSON(r, &req); err != nil {
		writeUsageError(w, http.StatusBadRequest, err)
		return
	}
	view, warnings, err := itemsService.Add(usageitems.AddRequest{
		Item:         req.item(),
		Enabled:      req.Enabled,
		Default:      req.Default,
		SkipValidate: req.SkipValidate,
		Strict:       req.Strict,
	})
	if err != nil {
		writeUsageError(w, usageStatusForError(err), err)
		return
	}
	writeUsageJSON(w, http.StatusOK, usageItemResult{Item: view, Warnings: warnings})
}

func handleUsageItemsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req usageItemUpdateRequest
	if err := decodeUsageJSON(r, &req); err != nil {
		writeUsageError(w, http.StatusBadRequest, err)
		return
	}
	view, warnings, err := itemsService.Update(usageitems.UpdateRequest{
		ID: req.ID,
		Update: usageitems.ItemUpdate{
			Label:   req.Label,
			Kind:    req.Kind,
			Home:    req.Home,
			APIURL:  req.APIURL,
			Enabled: req.Enabled,
		},
		SkipValidate: req.SkipValidate,
		Strict:       req.Strict,
	})
	if err != nil {
		writeUsageError(w, usageStatusForError(err), err)
		return
	}
	writeUsageJSON(w, http.StatusOK, usageItemResult{Item: view, Warnings: warnings})
}

func handleUsageItemsRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req usageItemRequest
	if err := decodeUsageJSON(r, &req); err != nil {
		writeUsageError(w, http.StatusBadRequest, err)
		return
	}
	if req.ID == "" {
		writeUsageError(w, http.StatusBadRequest, errMissingID)
		return
	}
	result, err := itemsService.Remove(req.ID)
	if err != nil {
		writeUsageError(w, usageStatusForError(err), err)
		return
	}
	writeUsageJSON(w, http.StatusOK, usageItemRemoveResult{
		Removed:  result.Removed,
		Default:  result.Default,
		Warnings: result.Warnings,
	})
}

func handleUsageItemsDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req usageItemDefaultRequest
	if err := decodeUsageJSON(r, &req); err != nil {
		writeUsageError(w, http.StatusBadRequest, err)
		return
	}
	list, err := itemsService.SetDefault(req.ID, req.Rotate)
	if err != nil {
		writeUsageError(w, usageStatusForError(err), err)
		return
	}
	writeUsageJSON(w, http.StatusOK, list)
}

// TestExported_SetItemsService points the package handlers at a service backed
// by a temporary registry, so tests exercise the real HTTP surface. It returns
// the service it replaced.
func TestExported_SetItemsService(svc *usageitems.Service) *usageitems.Service {
	prev := itemsService
	itemsService = svc
	return prev
}

type usageItemRequest struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Kind         string `json:"kind"`
	Home         string `json:"home"`
	APIURL       string `json:"api_url"`
	Enabled      *bool  `json:"enabled"`
	Default      bool   `json:"default"`
	SkipValidate bool   `json:"skip_validate"`
	Strict       bool   `json:"strict"`
}

func (req usageItemRequest) item() usageitems.Item {
	return usageitems.Item{
		ID:      req.ID,
		Label:   req.Label,
		Kind:    req.Kind,
		Home:    req.Home,
		APIURL:  req.APIURL,
		Enabled: true,
	}
}

type usageItemUpdateRequest struct {
	ID           string  `json:"id"`
	Label        *string `json:"label"`
	Kind         *string `json:"kind"`
	Home         *string `json:"home"`
	APIURL       *string `json:"api_url"`
	Enabled      *bool   `json:"enabled"`
	SkipValidate bool    `json:"skip_validate"`
	Strict       bool    `json:"strict"`
}

type usageItemDefaultRequest struct {
	ID     string `json:"id"`
	Rotate bool   `json:"rotate"`
}

type usageItemResult struct {
	Item     *usageitems.ItemView `json:"item,omitempty"`
	Warnings []string             `json:"warnings,omitempty"`
}

type usageItemRemoveResult struct {
	Removed  string   `json:"removed,omitempty"`
	Default  string   `json:"default,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func handleDebugLog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
	case http.MethodPut:
		var req debugSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := debuglog.SetEnabled(req.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		debuglog.Write(debuglog.Entry{
			Event: "debug_toggled",
			Labels: map[string]string{
				"component": "server",
				"phase":     "settings",
			},
			Fields: map[string]any{"enabled": req.Enabled},
		})
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
