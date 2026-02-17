package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

var (
	logFilesMu   sync.RWMutex
	logFiles     []LogFile
	logFilesOnce sync.Once
)

type LogFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func logFilesConfigPath() string {
	return filepath.Join(config.DataDir, "log-files.json")
}

func defaultLogFiles() []LogFile {
	return []LogFile{
		{Name: "server", Path: "ai-critic-server.log"},
		{Name: "keep-alive", Path: "ai-critic-server-keep-alive.log"},
	}
}

func LoadLogFiles() ([]LogFile, error) {
	logFilesMu.RLock()
	if logFiles != nil {
		logFilesMu.RUnlock()
		return copyLogFiles(logFiles), nil
	}
	logFilesMu.RUnlock()

	var loaded bool
	logFilesOnce.Do(func() {
		logFilesMu.Lock()
		defer logFilesMu.Unlock()

		if logFiles != nil {
			loaded = true
			return
		}

		data, err := os.ReadFile(logFilesConfigPath())
		if err != nil {
			if os.IsNotExist(err) {
				logFiles = defaultLogFiles()
				loaded = true
				return
			}
			return
		}

		var files []LogFile
		if err := json.Unmarshal(data, &files); err != nil {
			return
		}
		logFiles = files
		loaded = true
	})

	if !loaded {
		return nil, fmt.Errorf("failed to load log files")
	}
	return copyLogFiles(logFiles), nil
}

func GetLogFiles() ([]LogFile, error) {
	files, err := LoadLogFiles()
	if err != nil {
		return nil, err
	}
	return files, nil
}

func SaveLogFiles(files []LogFile) error {
	logFilesMu.Lock()
	defer logFilesMu.Unlock()

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(logFilesConfigPath(), data, 0644); err != nil {
		return err
	}

	logFiles = files
	return nil
}

func AddLogFile(name, path string) error {
	files, err := LoadLogFiles()
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.Name == name {
			return fmt.Errorf("log file with name %q already exists", name)
		}
	}

	files = append(files, LogFile{Name: name, Path: path})
	return SaveLogFiles(files)
}

func RemoveLogFile(name string) error {
	files, err := LoadLogFiles()
	if err != nil {
		return err
	}

	newFiles := make([]LogFile, 0)
	for _, f := range files {
		if f.Name != name {
			newFiles = append(newFiles, f)
		}
	}

	if len(newFiles) == len(files) {
		return fmt.Errorf("log file with name %q not found", name)
	}

	return SaveLogFiles(newFiles)
}

func GetLogFilePath(name string) (string, error) {
	files, err := LoadLogFiles()
	if err != nil {
		return "", err
	}

	for _, f := range files {
		if f.Name == name {
			return f.Path, nil
		}
	}

	return "", fmt.Errorf("log file with name %q not found", name)
}

func copyLogFiles(files []LogFile) []LogFile {
	result := make([]LogFile, len(files))
	copy(result, files)
	return result
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/logs/files", handleLogFiles)
	mux.HandleFunc("/api/logs/stream", handleLogStream)
}

func handleLogFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		files, err := GetLogFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Path == "" {
			http.Error(w, "name and path are required", http.StatusBadRequest)
			return
		}
		if err := AddLogFile(req.Name, req.Path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if err := RemoveLogFile(name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLogStream(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	linesStr := r.URL.Query().Get("lines")
	maxLines := 1000
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			maxLines = n
		}
	}

	fileName := r.URL.Query().Get("file")
	path := r.URL.Query().Get("path")

	var logPath string
	if path != "" {
		logPath = path
	} else if fileName != "" {
		var err error
		logPath, err = GetLogFilePath(fileName)
		if err != nil {
			sw.SendError(err.Error())
			return
		}
	} else {
		sw.SendError("either 'file' or 'path' query parameter is required")
		return
	}

	cmd := exec.Command("tail", fmt.Sprintf("-fn%d", maxLines), logPath)

	ctx := r.Context()
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("tail error: %v", err))
	}
}
