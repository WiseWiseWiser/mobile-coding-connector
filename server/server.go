package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/kool/pkgs/web"
	"github.com/xhd2015/lifelog-private/ai-critic/server/actions"
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents"
	"github.com/xhd2015/lifelog-private/ai-critic/server/auth"
	"github.com/xhd2015/lifelog-private/ai-critic/server/checkpoint"
	cloudflareSettings "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"
	"github.com/xhd2015/lifelog-private/ai-critic/server/exposedurls"
	"github.com/xhd2015/lifelog-private/ai-critic/server/fileupload"
	"github.com/xhd2015/lifelog-private/ai-critic/server/github"
	"github.com/xhd2015/lifelog-private/ai-critic/server/keepalive"
	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
	pfcloudflare "github.com/xhd2015/lifelog-private/ai-critic/server/portforward/providers/cloudflare"
	pflocaltunnel "github.com/xhd2015/lifelog-private/ai-critic/server/portforward/providers/localtunnel"
	"github.com/xhd2015/lifelog-private/ai-critic/server/projects"
	"github.com/xhd2015/lifelog-private/ai-critic/server/settings"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sshservers"
	"github.com/xhd2015/lifelog-private/ai-critic/server/subprocess"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tools"
)

var distFS embed.FS
var templateHTML string

func Init(fs embed.FS, tmpl string) {
	distFS = fs
	templateHTML = tmpl
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func EnsureFrontendDevServer(ctx context.Context) (chan struct{}, error) {
	// Check if 5173 is running
	fmt.Println("Frontend dev server (port 5173) not detected. Starting it...")
	cmd := exec.Command("bun", "run", "dev")
	cmd.Dir = "ai-critic-react/"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start frontend dev server: %v", err)
	}

	done := make(chan struct{})
	// Ensure sub-process is killed on context cancellation
	go func() {
		defer close(done)
		<-ctx.Done()
		if cmd.Process != nil {
			fmt.Println("Stopping frontend dev server...")
			// Kill the process group
			cmd.Process.Kill()
		}
	}()

	// Wait for port to be ready
	fmt.Print("Waiting for frontend server...")
	for i := 0; i < 30; i++ {
		if checkPort(5173) {
			fmt.Println(" Ready!")
			return done, nil
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()
	return nil, fmt.Errorf("frontend server failed to start within timeout")
}

func Serve(port int, dev bool) error {
	mux := http.NewServeMux()

	// Wrap with auth middleware - skip login, auth check, setup, ping, and public key endpoints
	handler := auth.Middleware(mux, []string{"/api/login", "/api/auth/check", "/api/auth/setup", "/ping", "/api/encrypt/public-key"})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute, // Long timeout for SSE streaming
		Handler:      handler,
	}

	if dev {
		if !checkPort(5173) {
			// Create context for managing subprocesses
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle signals to gracefully shutdown subprocesses
			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt, syscall.SIGTERM)
				<-c
				cancel()

				// wait the dev server to be closed
				if err := server.Close(); err != nil {
					fmt.Printf("Failed to close server: %v\n", err)
				}
			}()

			subProcessDone, err := EnsureFrontendDevServer(ctx)
			if err != nil {
				return err
			}
			if subProcessDone != nil {
				defer func() {
					fmt.Println("Waiting for frontend dev server to be closed...")
					<-subProcessDone
				}()
			}
		}

		err := ProxyDev(mux)
		if err != nil {
			return err
		}
	} else {
		err := Static(mux, StaticOptions{})
		if err != nil {
			return err
		}
	}

	err := RegisterAPI(mux)
	if err != nil {
		return err
	}

	fmt.Printf("Serving directory preview at http://localhost:%d\n", port)
	printTunnelHints(port)

	go func() {
		time.Sleep(1 * time.Second)
		web.OpenBrowser(fmt.Sprintf("http://localhost:%d", port))
	}()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	// Wait for either server error or shutdown signal
	select {
	case err := <-serverErr:
		return err
	case <-WaitForShutdown():
		// Graceful shutdown initiated
		fmt.Println("\nShutdown signal received, stopping server...")

		// Stop all port forwards (tunnels)
		pfManager := portforward.GetDefaultManager()
		for _, pf := range pfManager.List() {
			fmt.Printf("Stopping port forward for port %d...\n", pf.LocalPort)
			pfManager.Remove(pf.LocalPort)
		}

		// Stop all managed subprocesses
		fmt.Println("Stopping all managed subprocesses...")
		subprocess.GetManager().StopAll()

		// Shutdown HTTP server with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
			return err
		}

		fmt.Println("Server shutdown complete")
		return nil
	}
}

func ProxyDev(mux *http.ServeMux) error {
	targetURL, err := url.Parse("http://localhost:5173")
	if err != nil {
		return fmt.Errorf("invalid proxy target: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Proxy everything else to the frontend dev server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.Host = targetURL.Host
		proxy.ServeHTTP(w, r)
	})
	return nil
}

type StaticOptions struct {
	IndexHtml string // Custom HTML content to serve instead of embedded index.html
}

func Static(mux *http.ServeMux, opts StaticOptions) error {
	// Serve static files from the embedded React build
	reactFileSystem, err := fs.Sub(distFS, "ai-critic-react/dist")
	if err != nil {
		return fmt.Errorf("failed to create react file system: %v", err)
	}

	// Create sub-filesystem for assets
	assetsFileSystem, err := fs.Sub(reactFileSystem, "assets")
	if err != nil {
		return fmt.Errorf("failed to create assets file system: %v", err)
	}

	// Serve React assets from /assets/ path with proper MIME types

	// Serve index.css and index.js from assets with pattern matching
	mux.HandleFunc("/assets/index.css", func(w http.ResponseWriter, r *http.Request) {
		serveAssetWithPattern(w, r, assetsFileSystem, "index.css", "index-", ".css", "text/css")
	})
	mux.HandleFunc("/assets/index.js", func(w http.ResponseWriter, r *http.Request) {
		serveAssetWithPattern(w, r, assetsFileSystem, "index.js", "index-", ".js", "application/javascript")
	})

	mux.Handle("/assets/", http.StripPrefix("/assets/", &mimeTypeHandler{http.FileServer(http.FS(assetsFileSystem))}))
	// Serve React static files from root
	mux.Handle("/ai-critic.svg", &mimeTypeHandler{http.FileServer(http.FS(reactFileSystem))})
	// Serve PWA manifest.json
	mux.Handle("/manifest.json", &mimeTypeHandler{http.FileServer(http.FS(reactFileSystem))})

	// Serve the main HTML page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		// Use custom IndexHtml if provided
		if opts.IndexHtml != "" {
			w.Write([]byte(opts.IndexHtml))
			return
		}

		// Otherwise, serve embedded index.html
		indexFile, err := reactFileSystem.Open("index.html")
		if err != nil {
			http.Error(w, "Failed to load index.html", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()

		content, err := io.ReadAll(indexFile)
		if err != nil {
			http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
			return
		}

		w.Write(content)
	})
	return nil
}

func RegisterAPI(mux *http.ServeMux) error {
	// Initialize tool resolution: load user extra paths from terminal config
	// so that all subsequent LookPath calls respect them.
	if termCfg, err := terminal.LoadConfig(); err == nil && len(termCfg.ExtraPaths) > 0 {
		tool_resolve.SetUserExtraPaths(termCfg.ExtraPaths)
	}

	// ping
	mux.HandleFunc("/ping", handlePing)

	// auth API (login)
	auth.RegisterAPI(mux)

	// code review API
	registerReviewAPI(mux)

	// terminal API
	terminal.RegisterAPI(mux)

	// port forwarding: register providers and API
	portforward.RegisterDefaultProvider(&pflocaltunnel.Provider{})
	portforward.RegisterDefaultProvider(&pfcloudflare.QuickProvider{})
	portforward.RegisterDefaultProvider(&pfcloudflare.OwnedProvider{})

	// Register cloudflare_tunnel provider from config if available
	if cfg := config.Get(); cfg != nil {
		for _, provCfg := range cfg.PortForwarding.Providers {
			if !provCfg.IsEnabled() {
				continue
			}
			if provCfg.Type == portforward.ProviderCloudflareTunnel && provCfg.Cloudflare != nil {
				portforward.RegisterDefaultProvider(
					pfcloudflare.NewTunnelProvider(*provCfg.Cloudflare),
				)
			}
		}
	}

	portforward.RegisterAPI(mux)

	// GitHub API
	github.RegisterAPI(mux)

	// Encryption API (public key for frontend)
	encrypt.RegisterAPI(mux)

	// Projects API
	projects.RegisterAPI(mux)

	// Agents API
	agents.RegisterAPI(mux)

	// Checkpoint API
	checkpoint.RegisterAPI(mux)

	// Actions API
	actions.RegisterAPI(mux)

	// SSH Servers API
	sshservers.RegisterAPI(mux)

	// Tools diagnostics API
	tools.RegisterAPI(mux)

	// Cloudflare settings API
	cloudflareSettings.RegisterAPI(mux)

	// File upload API
	fileupload.RegisterAPI(mux)

	// Domains API
	domains.RegisterAPI(mux)

	// Settings export/import API
	settings.RegisterAPI(mux)

	// Exposed URLs API
	exposedurls.RegisterAPI(mux)

	// Keep-alive proxy API
	keepalive.RegisterAPI(mux)

	// Server config API
	mux.HandleFunc("/api/server/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetServerConfig(w, r)
		case http.MethodPost:
			SetServerConfig(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// AI config API
	registerAIConfigAPI(mux)

	// Build from source API
	registerBuildAPI(mux)
	// Graceful shutdown endpoint
	mux.HandleFunc("/api/shutdown", shutdownHandler)

	return nil
}

// printTunnelHints prints commands to expose the server via temporary tunnels.
func printTunnelHints(port int) {
	fmt.Println()
	fmt.Println("To expose this server to the internet via a temporary tunnel:")
	fmt.Println()

	// Option 1: Cloudflare
	fmt.Println("  # Option 1: Cloudflare Quick Tunnel")
	if hint := tools.GetInstallHint("cloudflared"); hint != "" {
		fmt.Printf("  # Install: %s\n", hint)
	}
	fmt.Printf("  cloudflared tunnel --url http://localhost:%d\n", port)
	fmt.Println()

	// Option 2: localtunnel
	fmt.Println("  # Option 2: localtunnel")
	if hint := tools.GetInstallHint("node"); hint != "" {
		fmt.Printf("  # Install Node.js: %s\n", hint)
	}
	fmt.Printf("  npx localtunnel --port %d\n", port)
	fmt.Println()
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// mimeTypeHandler wraps an http.Handler and sets proper MIME types
type mimeTypeHandler struct {
	handler http.Handler
}

func (h *mimeTypeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set MIME type based on file extension
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		// Use Go's built-in MIME type detection for other files
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
	}

	// Call the wrapped handler
	h.handler.ServeHTTP(w, r)
}

// serveAssetWithPattern finds and serves the first available file matching the given exact match or prefix and suffix
func serveAssetWithPattern(w http.ResponseWriter, r *http.Request, assetsFS fs.FS, exactMatch, prefix, suffix, contentType string) {
	// First try exact match
	if _, err := fs.Stat(assetsFS, exactMatch); err == nil {
		serveAssetFile(w, r, assetsFS, exactMatch, contentType)
		return
	}

	// Then try pattern matching with prefix and suffix
	entries, err := fs.ReadDir(assetsFS, ".")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), suffix) {
			serveAssetFile(w, r, assetsFS, entry.Name(), contentType)
			return
		}
	}

	// No matching file found
	http.NotFound(w, r)
}

// serveAssetFile serves a specific file from the assets filesystem
func serveAssetFile(w http.ResponseWriter, r *http.Request, assetsFS fs.FS, filename string, contentType string) {
	file, err := assetsFS.Open(filename)
	if err != nil {
		http.Error(w, "Failed to open asset file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read asset file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(content)
}

// checkPortAvailable checks if a port is available
func checkPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// FindAvailablePort finds a port starting from startPort
func FindAvailablePort(startPort int, maxAttempts int) (int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		if checkPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}

// ---- Build from Source API ----

// registerBuildAPI registers the build endpoints in the main server.
// These endpoints run in the main server (not the keep-alive daemon) to ensure
// proper environment setup with all PATH additions from tool_resolve.
func registerBuildAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/build/buildable-projects", handleBuildableProjectsMain)
	mux.HandleFunc("/api/build/build-next", handleBuildNextMain)
}

// buildableProject represents a project that can be built
type buildableProject struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	HasGoMod       bool   `json:"has_go_mod"`
	HasBuildScript bool   `json:"has_build_script"`
}

// findBuildableProjects scans all projects and finds those that can be built.
// Also checks the server-configured project directory.
func findBuildableProjects() ([]buildableProject, error) {
	var buildable []buildableProject
	seenDirs := make(map[string]bool)

	// First, check the server-configured project directory
	serverProjectDir := GetEffectiveProjectDir()
	if serverProjectDir != "" {
		if project := checkBuildableProject(serverProjectDir, "server", "Server Project"); project != nil {
			buildable = append(buildable, *project)
			seenDirs[serverProjectDir] = true
		}
	}

	// Then check projects from projects.json
	projectsFile := config.ProjectsFile
	data, err := os.ReadFile(projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return buildable, nil
		}
		return nil, err
	}

	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Dir  string `json:"dir"`
	}
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Dir == "" || seenDirs[p.Dir] {
			continue
		}

		if project := checkBuildableProject(p.Dir, p.ID, p.Name); project != nil {
			buildable = append(buildable, *project)
			seenDirs[p.Dir] = true
		}
	}

	return buildable, nil
}

// checkBuildableProject checks if a directory is a buildable project.
// Returns nil if not buildable.
func checkBuildableProject(dir, id, name string) *buildableProject {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	// Check for go.mod
	hasGoMod := false
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		hasGoMod = true
	}

	// Check for build script
	hasBuildScript := false
	buildScriptPath := filepath.Join(dir, "script", "server", "build", "for-linux-amd64")
	if _, err := os.Stat(buildScriptPath); err == nil {
		hasBuildScript = true
	}

	if hasGoMod && hasBuildScript {
		return &buildableProject{
			ID:             id,
			Name:           name,
			Dir:            dir,
			HasGoMod:       true,
			HasBuildScript: true,
		}
	}

	return nil
}

// handleBuildableProjectsMain returns the list of projects that can be built from source.
func handleBuildableProjectsMain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	buildable, err := findBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildable)
}

// handleBuildNextMain builds the next binary from a project source with SSE streaming.
// This runs in the main server to ensure proper environment with PATH additions.
func handleBuildNextMain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Find buildable projects
	buildable, err := findBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the requested project or use the first available
	var project *buildableProject
	if req.ProjectID != "" {
		for i := range buildable {
			if buildable[i].ID == req.ProjectID {
				project = &buildable[i]
				break
			}
		}
	} else if len(buildable) > 0 {
		project = &buildable[0]
	}

	if project == nil {
		http.Error(w, "no buildable project found", http.StatusBadRequest)
		return
	}

	// Get the upload target path (next binary)
	binPath, err := os.Executable()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get executable path: %v", err), http.StatusInternalServerError)
		return
	}

	dir := filepath.Dir(binPath)
	currentBase, currentVersion := parseBinVersion(binPath)
	nextVersion := currentVersion + 1
	newName := fmt.Sprintf("%s-v%d", currentBase, nextVersion)
	destPath := filepath.Join(dir, newName)

	// Create SSE writer
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Log build start
	sw.SendLog(fmt.Sprintf("Building next binary (v%d) from project %s...", nextVersion, project.Name))
	sw.SendLog(fmt.Sprintf("Target: %s", destPath))

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		sw.SendError(fmt.Sprintf("Failed to create destination directory: %v", err))
		return
	}

	// Run build script using go run to ensure environment variables are inherited
	// Use the proper environment with all PATH additions
	cmd := exec.Command("go", "run", "./script/server/build/for-linux-amd64", "-o", destPath)
	cmd.Dir = project.Dir
	// Set up environment with all extra paths (same as terminal)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)

	err = sw.StreamCmd(cmd)
	if err != nil {
		sw.SendError(fmt.Sprintf("Build failed: %v", err))
		return
	}

	// Make binary executable
	if err := os.Chmod(destPath, 0755); err != nil {
		sw.SendError(fmt.Sprintf("Failed to chmod binary: %v", err))
		return
	}

	// Get file size
	info, err := os.Stat(destPath)
	if err != nil {
		sw.SendError(fmt.Sprintf("Failed to stat binary: %v", err))
		return
	}

	// Log success
	sw.SendLog(fmt.Sprintf("Build successful: %s (%d bytes)", destPath, info.Size()))

	// Send done event with result data
	sw.SendDone(map[string]string{
		"success":      "true",
		"message":      fmt.Sprintf("Built %s (%s) v%d", newName, project.Name, nextVersion),
		"binary_path":  destPath,
		"binary_name":  newName,
		"version":      strconv.Itoa(nextVersion),
		"size":         strconv.FormatInt(info.Size(), 10),
		"project_name": project.Name,
	})
}

// parseBinVersion extracts the base name and version from a binary path.
func parseBinVersion(binPath string) (baseName string, version int) {
	name := filepath.Base(binPath)

	// Match -vN suffix
	if idx := strings.LastIndex(name, "-v"); idx != -1 && idx < len(name)-2 {
		versionStr := name[idx+2:]
		if v, err := strconv.Atoi(versionStr); err == nil {
			return name[:idx], v
		}
	}

	return name, 0
}

// shutdownHandler handles graceful shutdown of the server
func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Start shutdown in background
	go func() {
		// Give HTTP response time to be sent
		time.Sleep(2 * time.Second)
		// Trigger graceful shutdown
		ShutdownServer()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "shutting_down",
		"message":   "Server will shutdown in 2 seconds",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// globalShutdownChan is used to signal server shutdown
var globalShutdownChan = make(chan struct{})

// ShutdownServer initiates server shutdown
func ShutdownServer() {
	close(globalShutdownChan)
}

// WaitForShutdown returns a channel that will be closed when shutdown is requested
func WaitForShutdown() <-chan struct{} {
	return globalShutdownChan
}
