package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// TunnelInfo represents a Cloudflare tunnel.
type TunnelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at,omitempty"`
	Connections []any  `json:"connections,omitempty"`
}

// CertFileInfo describes a cloudflared credential file.
type CertFileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// StatusResponse is the response from GET /api/cloudflare/status.
type StatusResponse struct {
	Installed     bool           `json:"installed"`
	Authenticated bool           `json:"authenticated"`
	Error         string         `json:"error,omitempty"`
	CertFiles     []CertFileInfo `json:"cert_files,omitempty"`
}

// RegisterAPI registers cloudflare settings API endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/cloudflare/status", handleStatus)
	mux.HandleFunc("/api/cloudflare/login", handleLogin)
	mux.HandleFunc("/api/cloudflare/tunnels", handleTunnels)
	mux.HandleFunc("/api/cloudflare/download", handleDownload)
	mux.HandleFunc("/api/cloudflare/upload", handleUpload)
}

// cloudflaredDir returns the path to the cloudflared config directory.
func cloudflaredDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cloudflared")
	}
	return ""
}

// ListCertFiles discovers cloudflared credential files.
func ListCertFiles() []CertFileInfo {
	dir := cloudflaredDir()
	if dir == "" {
		return nil
	}

	var files []CertFileInfo
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Include cert.pem and tunnel credential JSON files
		if name == "cert.pem" || strings.HasSuffix(name, ".json") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, CertFileInfo{
				Name: name,
				Path: filepath.Join(dir, name),
				Size: info.Size(),
			})
		}
	}
	return files
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := CheckStatus()
	writeJSON(w, resp)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	sw.SendLog("Starting cloudflared login...")
	sw.SendLog("A browser window should open. If running in a container, copy the URL below and open it manually.")

	cmd := exec.Command("cloudflared", "tunnel", "login")
	urlRe := regexp.MustCompile(`https://dash\.cloudflare\.com/argotunnel\S+`)
	err := sw.StreamCmdFunc(cmd, func(line string) bool {
		// Detect auth URL and send as a special event
		if m := urlRe.FindString(line); m != "" {
			sw.Send(map[string]string{"type": "auth_url", "url": m})
		}
		return true // always also send as log
	})
	if err != nil {
		sw.SendError(fmt.Sprintf("Login failed: %v", err))
	} else {
		sw.SendDone(map[string]string{"message": "Login successful! You are now authenticated."})
	}
}

func handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listTunnels(w)
	case http.MethodPost:
		createTunnel(w, r)
	case http.MethodDelete:
		deleteTunnel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listTunnels(w http.ResponseWriter) {
	out, err := exec.Command("cloudflared", "tunnel", "list", "--output", "json").CombinedOutput()
	if err != nil {
		http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
		return
	}

	// Parse and re-encode to ensure valid JSON
	var tunnels []TunnelInfo
	if err := json.Unmarshal(out, &tunnels); err != nil {
		// If parse fails, return raw output
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	writeJSON(w, tunnels)
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	out, err := exec.Command("cloudflared", "tunnel", "create", name).CombinedOutput()
	if err != nil {
		http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"message": strings.TrimSpace(string(out)),
	})
}

func deleteTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	out, err := exec.Command("cloudflared", "tunnel", "delete", name).CombinedOutput()
	if err != nil {
		http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"message": strings.TrimSpace(string(out)),
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	// Security: only allow downloading files from the .cloudflared directory
	dir := cloudflaredDir()
	if dir == "" {
		http.Error(w, "Could not determine cloudflared directory", http.StatusInternalServerError)
		return
	}

	// Prevent path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(dir, name)
	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	http.ServeFile(w, r, filePath)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := cloudflaredDir()
	if dir == "" {
		http.Error(w, "Could not determine cloudflared directory", http.StatusInternalServerError)
		return
	}

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0700); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	uploaded := 0
	for _, fileHeaders := range r.MultipartForm.File {
		for _, fh := range fileHeaders {
			name := filepath.Base(fh.Filename)
			// Only allow cert.pem and .json files
			if name != "cert.pem" && !strings.HasSuffix(name, ".json") {
				continue
			}
			// Prevent path traversal
			if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
				continue
			}

			src, err := fh.Open()
			if err != nil {
				continue
			}

			dstPath := filepath.Join(dir, name)
			dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				src.Close()
				continue
			}

			if _, err := io.Copy(dst, src); err != nil {
				src.Close()
				dst.Close()
				continue
			}
			src.Close()
			dst.Close()
			uploaded++
		}
	}

	if uploaded == 0 {
		http.Error(w, "No valid files uploaded. Only cert.pem and .json files are accepted.", http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"message":  fmt.Sprintf("Uploaded %d file(s) successfully", uploaded),
		"uploaded": uploaded,
	})
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
