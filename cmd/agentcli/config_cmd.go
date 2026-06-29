package agentcli

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
)

//go:embed config.html
var configHTMLTemplate string

type configPageData struct {
	Title             string
	ConfigPath        string
	CLIName           string
	PortFlagNote      bool
	DefaultServerNote template.HTML
	ServerPlaceholder string
}

func configHelpFor(p Profile) string {
	extra := ""
	if p.SupportsPortFlag {
		extra = fmt.Sprintf(`
When other commands run without --server or --port, the default domain's server
and token are used automatically (built-in default: http://localhost:%d).`,
			p.DefaultPort)
	} else {
		extra = `
When other commands run without --server, the default domain's server and token
are used automatically.`
	}
	return fmt.Sprintf(`Usage: %s config

Open a local web page to manage saved server domains and pick a default one.%s

Examples:
  %s config
`, p.Name, extra, p.Name)
}

func configPageDataFor(p Profile) configPageData {
	data := configPageData{
		Title:             "Remote Agent Config",
		ConfigPath:        "~/.ai-critic/remote-agent-config.json",
		CLIName:           p.Name,
		ServerPlaceholder: "https://host.example.com",
	}
	if p.Name == "local-agent" {
		data.Title = "Local Agent Config"
		data.ConfigPath = "~/.ai-critic/local-agent-config.json"
		data.PortFlagNote = true
		data.ServerPlaceholder = fmt.Sprintf("http://localhost:%d", p.DefaultPort)
		data.DefaultServerNote = template.HTML(fmt.Sprintf(
			"With no saved default, commands use <code>http://localhost:%d</code>.",
			p.DefaultPort,
		))
	}
	return data
}

func renderConfigHTML(p Profile) (string, error) {
	tmpl, err := template.New("config").Parse(configHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse config template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, configPageDataFor(p)); err != nil {
		return "", fmt.Errorf("render config template: %w", err)
	}
	return buf.String(), nil
}

func runConfig(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(configHelpFor(active))
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("config takes no arguments, got %v; see '%s config --help'", args, active.Name)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local listener: %w", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	shutdown := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleConfigIndex)
	mux.HandleFunc("/api/config", handleConfigAPI)
	mux.HandleFunc("/api/test", handleConfigTest)
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
		go func() {
			time.Sleep(100 * time.Millisecond)
			close(shutdown)
		}()
	})

	server := &http.Server{Handler: mux}

	go func() {
		_ = server.Serve(listener)
	}()

	fmt.Printf("Config UI running at %s\n", url)
	fmt.Println("Press Ctrl+C to exit after saving.")

	if err := openBrowser(url); err != nil {
		fmt.Printf("(could not open browser: %v)\n", err)
	}

	<-shutdown
	fmt.Println("Config saved. Shutting down.")
	_ = server.Close()
	return nil
}

func handleConfigIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	html, err := renderConfigHTML(active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleConfigAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := loadConfig()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if cfg == nil {
			cfg = &agentConfig{}
		}
		writeJSON(w, cfg)
	case http.MethodPost:
		var req agentConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		normalizeIncomingConfig(&req)
		if err := saveConfig(&req); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// normalizeIncomingConfig trims whitespace/trailing slashes and clears the
// default selection when it references a server no longer in Domains.
func normalizeIncomingConfig(cfg *agentConfig) {
	cfg.Default = strings.TrimRight(strings.TrimSpace(cfg.Default), "/")
	cleaned := make([]domainConfig, 0, len(cfg.Domains))
	seen := map[string]bool{}
	for _, d := range cfg.Domains {
		server := strings.TrimRight(strings.TrimSpace(d.Server), "/")
		if server == "" || seen[server] {
			continue
		}
		seen[server] = true
		cleaned = append(cleaned, domainConfig{
			Server: server,
			Token:  strings.TrimSpace(d.Token),
		})
	}
	cfg.Domains = cleaned
	if !seen[cfg.Default] {
		cfg.Default = ""
	}
}

func handleConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Server string `json:"server"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	server := strings.TrimRight(strings.TrimSpace(req.Server), "/")
	if server == "" {
		writeJSONError(w, http.StatusBadRequest, "server is required")
		return
	}

	cli := client.New(server, strings.TrimSpace(req.Token))
	if err := cli.CheckAuth(); err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
