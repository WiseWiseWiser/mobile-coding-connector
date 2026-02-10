package settings

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/auth"
	cloudflareSettings "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

// RegisterAPI registers settings export/import endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/settings/export", handleExport)
	mux.HandleFunc("/api/settings/import", handleImport)

	// Zip-based export/import
	mux.HandleFunc("/api/settings/export-zip", handleExportZip)
	mux.HandleFunc("/api/settings/import-zip/preview", handleImportZipPreview)
	mux.HandleFunc("/api/settings/import-zip/confirm", handleImportZipConfirm)
	mux.HandleFunc("/api/settings/import-zip/browser-data", handleImportZipBrowserData)
}

// ---- Export Types ----

type EncryptionKeysExport struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type WebDomainsExport struct {
	Domains    []domains.DomainEntry `json:"domains"`
	TunnelName string                `json:"tunnel_name"`
}

type CloudflareFileExport struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"content_base64"`
}

type CloudflareAuthExport struct {
	Files []CloudflareFileExport `json:"files"`
}

type CredentialsExport struct {
	Tokens []string `json:"tokens"`
}

type TerminalConfigExport struct {
	ExtraPaths []string `json:"extra_paths"`
	Shell      string   `json:"shell,omitempty"`
	ShellFlags []string `json:"shell_flags,omitempty"`
}

type ExportSections struct {
	EncryptionKeys *EncryptionKeysExport `json:"encryption_keys,omitempty"`
	WebDomains     *WebDomainsExport     `json:"web_domains,omitempty"`
	CloudflareAuth *CloudflareAuthExport `json:"cloudflare_auth,omitempty"`
	Credentials    *CredentialsExport    `json:"credentials,omitempty"`
	TerminalConfig *TerminalConfigExport `json:"terminal_config,omitempty"`
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sections := r.URL.Query()["section"]
	sectionSet := make(map[string]bool, len(sections))
	for _, s := range sections {
		sectionSet[s] = true
	}

	result := ExportSections{}

	if sectionSet["encryption_keys"] {
		data, err := exportEncryptionKeys()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("encryption_keys: %v", err))
			return
		}
		result.EncryptionKeys = data
	}

	if sectionSet["web_domains"] {
		data, err := exportWebDomains()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("web_domains: %v", err))
			return
		}
		result.WebDomains = data
	}

	if sectionSet["cloudflare_auth"] {
		data, err := exportCloudflareAuth()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("cloudflare_auth: %v", err))
			return
		}
		result.CloudflareAuth = data
	}

	if sectionSet["credentials"] {
		data, err := exportCredentials()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("credentials: %v", err))
			return
		}
		result.Credentials = data
	}

	if sectionSet["terminal_config"] {
		data, err := exportTerminalConfig()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("terminal_config: %v", err))
			return
		}
		result.TerminalConfig = data
	}

	writeJSON(w, result)
}

// ---- Import ----

type ImportSections struct {
	EncryptionKeys *EncryptionKeysExport `json:"encryption_keys,omitempty"`
	WebDomains     *WebDomainsExport     `json:"web_domains,omitempty"`
	CloudflareAuth *CloudflareAuthExport `json:"cloudflare_auth,omitempty"`
	Credentials    *CredentialsExport    `json:"credentials,omitempty"`
	TerminalConfig *TerminalConfigExport `json:"terminal_config,omitempty"`
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sections ImportSections
	if err := json.NewDecoder(r.Body).Decode(&sections); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if sections.EncryptionKeys != nil {
		if err := importEncryptionKeys(sections.EncryptionKeys); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("encryption_keys: %v", err))
			return
		}
	}

	if sections.WebDomains != nil {
		if err := importWebDomains(sections.WebDomains); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("web_domains: %v", err))
			return
		}
	}

	if sections.CloudflareAuth != nil {
		if err := importCloudflareAuth(sections.CloudflareAuth); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("cloudflare_auth: %v", err))
			return
		}
	}

	if sections.Credentials != nil {
		if err := importCredentialsData(sections.Credentials); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("credentials: %v", err))
			return
		}
	}

	if sections.TerminalConfig != nil {
		if err := importTerminalConfig(sections.TerminalConfig); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("terminal_config: %v", err))
			return
		}
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// ---- Encryption Keys ----

func exportEncryptionKeys() (*EncryptionKeysExport, error) {
	status := encrypt.GetKeyStatus()
	if !status.Exists {
		return &EncryptionKeysExport{}, nil
	}

	privData, err := os.ReadFile(status.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	pubData, err := os.ReadFile(status.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	return &EncryptionKeysExport{
		PrivateKey: string(privData),
		PublicKey:  string(pubData),
	}, nil
}

func importEncryptionKeys(data *EncryptionKeysExport) error {
	status := encrypt.GetKeyStatus()
	privPath := status.PrivateKeyPath
	pubPath := status.PublicKeyPath

	if data.PrivateKey != "" {
		if err := os.WriteFile(privPath, []byte(data.PrivateKey), 0600); err != nil {
			return fmt.Errorf("write private key: %w", err)
		}
	}
	if data.PublicKey != "" {
		if err := os.WriteFile(pubPath, []byte(data.PublicKey), 0644); err != nil {
			return fmt.Errorf("write public key: %w", err)
		}
	}
	return nil
}

// ---- Web Domains ----

func exportWebDomains() (*WebDomainsExport, error) {
	cfg, err := domains.LoadDomains()
	if err != nil {
		return nil, err
	}
	return &WebDomainsExport{
		Domains:    cfg.Domains,
		TunnelName: cfg.TunnelName,
	}, nil
}

func importWebDomains(data *WebDomainsExport) error {
	cfg := &domains.DomainsConfig{
		Domains:    data.Domains,
		TunnelName: data.TunnelName,
	}
	return domains.SaveDomains(cfg)
}

// ---- Cloudflare Auth Files ----

func cloudflaredDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cloudflared")
	}
	return ""
}

// opencodeConfigDir returns the opencode config directory (~/.local/share/opencode).
func opencodeConfigDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "opencode")
	}
	return ""
}

func exportCloudflareAuth() (*CloudflareAuthExport, error) {
	certFiles := cloudflareSettings.ListCertFiles()
	var files []CloudflareFileExport
	for _, cf := range certFiles {
		data, err := os.ReadFile(cf.Path)
		if err != nil {
			continue
		}
		files = append(files, CloudflareFileExport{
			Name:          cf.Name,
			ContentBase64: base64.StdEncoding.EncodeToString(data),
		})
	}
	return &CloudflareAuthExport{Files: files}, nil
}

func importCloudflareAuth(data *CloudflareAuthExport) error {
	dir := cloudflaredDir()
	if dir == "" {
		return fmt.Errorf("cannot determine cloudflared directory")
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create cloudflared dir: %w", err)
	}

	for _, f := range data.Files {
		// Security: only allow cert.pem and .json files
		name := filepath.Base(f.Name)
		if name != "cert.pem" && !strings.HasSuffix(name, ".json") {
			continue
		}
		if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
			continue
		}

		content, err := base64.StdEncoding.DecodeString(f.ContentBase64)
		if err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}

		dstPath := filepath.Join(dir, name)
		if err := os.WriteFile(dstPath, content, 0600); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// ---- Credentials ----

func exportCredentials() (*CredentialsExport, error) {
	tokens, err := auth.ExportCredentials()
	if err != nil {
		return nil, err
	}
	if tokens == nil {
		tokens = []string{}
	}
	return &CredentialsExport{Tokens: tokens}, nil
}

func importCredentialsData(data *CredentialsExport) error {
	return auth.ImportCredentials(data.Tokens)
}

// ---- Terminal Config ----

func exportTerminalConfig() (*TerminalConfigExport, error) {
	cfg, err := terminal.LoadConfig()
	if err != nil {
		return nil, err
	}
	paths := cfg.ExtraPaths
	if paths == nil {
		paths = []string{}
	}
	return &TerminalConfigExport{
		ExtraPaths: paths,
		Shell:      cfg.Shell,
		ShellFlags: cfg.ShellFlags,
	}, nil
}

func importTerminalConfig(data *TerminalConfigExport) error {
	cfg := &terminal.TerminalConfig{
		ExtraPaths: data.ExtraPaths,
		Shell:      data.Shell,
		ShellFlags: data.ShellFlags,
	}
	return terminal.SaveConfig(cfg)
}

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
