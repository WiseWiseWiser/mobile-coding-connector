package settings

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

func handleExportZip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If POST, read optional browser-data JSON from body
	var browserData []byte
	if r.Method == http.MethodPost {
		var err error
		browserData, err = io.ReadAll(io.LimitReader(r.Body, 1<<20)) // max 1MB
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("ai-critic-settings-%s.zip", timestamp)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	zw := zip.NewWriter(w)
	defer zw.Close()

	// 1. Add all files from .ai-critic/ directory, using "ai-critic" as the zip prefix
	if err := addDirToZipWithPrefix(zw, config.DataDir, zipPrefixAICritic[:len(zipPrefixAICritic)-1]); err != nil {
		// Headers already sent, can't change status code. Log and continue.
		fmt.Fprintf(os.Stderr, "export-zip: error adding %s dir: %v\n", config.DataDir, err)
	}

	// 2. Add cloudflare files from ~/.cloudflared/ into cloudflare/ prefix in the zip
	cfDir := cloudflaredDir()
	if cfDir != "" {
		if err := addCloudflareFilesToZip(zw, cfDir); err != nil {
			fmt.Fprintf(os.Stderr, "export-zip: error adding cloudflare files: %v\n", err)
		}
	}

	// 3. Add browser-data.json if provided
	if len(browserData) > 0 {
		fw, err := zw.Create("browser-data.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "export-zip: error creating browser-data.json: %v\n", err)
			return
		}
		if _, err := fw.Write(browserData); err != nil {
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
