package settings

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type exportZipRequest struct {
	IncludeAICritic   bool            `json:"include_ai_critic"`
	IncludeCloudflare bool            `json:"include_cloudflare"`
	IncludeOpencode   bool            `json:"include_opencode"`
	BrowserData       json.RawMessage `json:"browser_data"`
}

func handleExportZip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request options
	var req exportZipRequest
	// Default to including all categories
	req.IncludeAICritic = true
	req.IncludeCloudflare = true
	req.IncludeOpencode = true

	if r.Method == http.MethodGet {
		// Parse query parameters
		if r.URL.Query().Get("include_ai_critic") == "false" {
			req.IncludeAICritic = false
		}
		if r.URL.Query().Get("include_cloudflare") == "false" {
			req.IncludeCloudflare = false
		}
		if r.URL.Query().Get("include_opencode") == "false" {
			req.IncludeOpencode = false
		}
	} else if r.Method == http.MethodPost {
		// Parse JSON body
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // max 1MB
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				// If parsing fails, try to parse as just browser data for backwards compatibility
				req.BrowserData = body
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("ai-critic-settings-%s.zip", timestamp)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	zw := zip.NewWriter(w)
	defer zw.Close()

	// 1. Add all files from .ai-critic/ directory, using "ai-critic" as the zip prefix
	if req.IncludeAICritic {
		if err := addDirToZipWithPrefix(zw, config.DataDir, zipPrefixAICritic[:len(zipPrefixAICritic)-1]); err != nil {
			// Headers already sent, can't change status code. Log and continue.
			fmt.Fprintf(os.Stderr, "export-zip: error adding %s dir: %v\n", config.DataDir, err)
		}
	}

	// 2. Add cloudflare files from ~/.cloudflared/ into cloudflare/ prefix in the zip
	if req.IncludeCloudflare {
		cfDir := cloudflaredDir()
		if cfDir != "" {
			if err := addCloudflareFilesToZip(zw, cfDir); err != nil {
				fmt.Fprintf(os.Stderr, "export-zip: error adding cloudflare files: %v\n", err)
			}
		}
	}

	// 3. Add opencode files from ~/.local/share/opencode/ into opencode/ prefix in the zip
	if req.IncludeOpencode {
		opencodeDir := opencodeConfigDir()
		if opencodeDir != "" {
			if err := addOpencodeFilesToZip(zw, opencodeDir); err != nil {
				fmt.Fprintf(os.Stderr, "export-zip: error adding opencode files: %v\n", err)
			}
		}
	}

	// 4. Add browser-data.json if provided
	if len(req.BrowserData) > 0 {
		fw, err := zw.Create("browser-data.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "export-zip: error creating browser-data.json: %v\n", err)
			return
		}
		if _, err := fw.Write(req.BrowserData); err != nil {
			fmt.Fprintf(os.Stderr, "export-zip: error writing browser-data.json: %v\n", err)
		}
	}
}

// addDirToZipWithPrefix recursively adds all files under srcDir into the zip writer,
// using zipPrefix as the top-level directory name in the zip.
// e.g. srcDir=".ai-critic", zipPrefix="ai-critic" maps ".ai-critic/enc-key" -> "ai-critic/enc-key"
func addDirToZipWithPrefix(zw *zip.Writer, srcDir string, zipPrefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// Use forward slashes for zip paths
		zipPath := filepath.ToSlash(filepath.Join(zipPrefix, relPath))

		return addFileToZip(zw, path, zipPath)
	})
}

// addCloudflareFilesToZip adds cloudflare credential files (cert.pem, *.json)
// from the cloudflared directory into a "cloudflare/" prefix in the zip.
func addCloudflareFilesToZip(zw *zip.Writer, cfDir string) error {
	entries, err := os.ReadDir(cfDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only include cert.pem and .json files
		if name != "cert.pem" && !strings.HasSuffix(name, ".json") {
			continue
		}

		srcPath := filepath.Join(cfDir, name)
		zipPath := "cloudflare/" + name
		if err := addFileToZip(zw, srcPath, zipPath); err != nil {
			return err
		}
	}
	return nil
}

// addOpencodeFilesToZip adds opencode config files (auth.json, settings.json, etc.)
// from the opencode directory into a "opencode/" prefix in the zip.
// It also adds plugins from ~/.config/opencode/plugins/ into "opencode/plugins/".
// It also adds opencode.jsonc from ~/.config/opencode/ into "opencode/opencode.jsonc".
func addOpencodeFilesToZip(zw *zip.Writer, opencodeDir string) error {
	// Add config files from ~/.local/share/opencode/
	entries, err := os.ReadDir(opencodeDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Include all JSON files (auth.json, settings.json, etc.)
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		srcPath := filepath.Join(opencodeDir, name)
		zipPath := "opencode/" + name
		if err := addFileToZip(zw, srcPath, zipPath); err != nil {
			return err
		}
	}

	// Add opencode.jsonc from ~/.config/opencode/
	mainConfigDir := opencodeMainConfigDir()
	if mainConfigDir != "" {
		mainConfigPath := filepath.Join(mainConfigDir, "opencode.jsonc")
		if _, err := os.Stat(mainConfigPath); err == nil {
			zipPath := "opencode/opencode.jsonc"
			if err := addFileToZip(zw, mainConfigPath, zipPath); err != nil {
				return err
			}
		}
	}

	// Add plugins from ~/.config/opencode/plugins/
	pluginsDir := opencodePluginsDir()
	if pluginsDir == "" {
		return nil
	}
	pluginEntries, err := os.ReadDir(pluginsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range pluginEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		srcPath := filepath.Join(pluginsDir, name)
		zipPath := "opencode/plugins/" + name
		if err := addFileToZip(zw, srcPath, zipPath); err != nil {
			return err
		}
	}

	return nil
}

// addFileToZip adds a single file to the zip writer.
func addFileToZip(zw *zip.Writer, srcPath string, zipPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}

	fw, err := zw.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", zipPath, err)
	}

	if _, err := fw.Write(data); err != nil {
		return fmt.Errorf("write zip entry %s: %w", zipPath, err)
	}
	return nil
}
