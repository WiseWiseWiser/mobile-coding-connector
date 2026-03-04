package features

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type Feature struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
}

var mu sync.Mutex

func featuresDir(projectName string) string {
	return filepath.Join(config.ProjectsDir, projectName)
}

func featuresFile(projectName string) string {
	return filepath.Join(featuresDir(projectName), "features.json")
}

func loadFeatures(projectName string) ([]Feature, error) {
	data, err := os.ReadFile(featuresFile(projectName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var features []Feature
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, err
	}
	return features, nil
}

func saveFeatures(projectName string, features []Feature) error {
	if err := os.MkdirAll(featuresDir(projectName), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(features, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(featuresFile(projectName), data, 0644)
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 36) + fmt.Sprintf("%04x", rand.Intn(0xffff))
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/features", handleFeatures)
}

func handleFeatures(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")
	if projectName == "" {
		http.Error(w, "missing project parameter", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		features, err := loadFeatures(projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if features == nil {
			features = []Feature{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(features)

	case http.MethodPost:
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		features, err := loadFeatures(projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		feature := Feature{
			ID:          generateID(),
			Title:       req.Title,
			Description: req.Description,
			Status:      "draft",
			CreatedAt:   time.Now().UnixMilli(),
		}
		features = append([]Feature{feature}, features...)

		if err := saveFeatures(projectName, features); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(feature)

	case http.MethodDelete:
		featureID := r.URL.Query().Get("id")
		if featureID == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		features, err := loadFeatures(projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		found := false
		for i, f := range features {
			if f.ID == featureID {
				features = append(features[:i], features[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			http.Error(w, "feature not found", http.StatusNotFound)
			return
		}

		if err := saveFeatures(projectName, features); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodPatch:
		featureID := r.URL.Query().Get("id")
		if featureID == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}

		var req struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		features, err := loadFeatures(projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var updated *Feature
		for i, f := range features {
			if f.ID == featureID {
				if req.Title != nil {
					features[i].Title = *req.Title
				}
				if req.Description != nil {
					features[i].Description = *req.Description
				}
				updated = &features[i]
				break
			}
		}
		if updated == nil {
			http.Error(w, "feature not found", http.StatusNotFound)
			return
		}

		if err := saveFeatures(projectName, features); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
