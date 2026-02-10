package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// BuildableProject represents a project that can be built
type BuildableProject struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	HasGoMod       bool   `json:"has_go_mod"`
	HasBuildScript bool   `json:"has_build_script"`
}

// FindBuildableProjects scans all projects and finds those that can be built
func FindBuildableProjects() ([]BuildableProject, error) {
	projectsFile := config.ProjectsFile

	data, err := os.ReadFile(projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []BuildableProject{}, nil
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

	var buildable []BuildableProject
	for _, p := range projects {
		if p.Dir == "" {
			continue
		}

		// Check if directory exists
		info, err := os.Stat(p.Dir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Check for go.mod
		hasGoMod := false
		if _, err := os.Stat(filepath.Join(p.Dir, "go.mod")); err == nil {
			hasGoMod = true
		}

		// Check for build script
		hasBuildScript := false
		buildScriptPath := filepath.Join(p.Dir, "script", "server", "build", "for-linux-amd64")
		if _, err := os.Stat(buildScriptPath); err == nil {
			hasBuildScript = true
		}

		if hasGoMod && hasBuildScript {
			buildable = append(buildable, BuildableProject{
				ID:             p.ID,
				Name:           p.Name,
				Dir:            p.Dir,
				HasGoMod:       true,
				HasBuildScript: true,
			})
		}
	}

	return buildable, nil
}

// StreamLogs streams the server log via tail -fn100
func StreamLogs(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	linesStr := r.URL.Query().Get("lines")
	maxLines := "100"
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			maxLines = strconv.Itoa(n)
		}
	}

	logPath := config.ServerLogFile
	cmd := exec.Command("tail", "-fn"+maxLines, logPath)

	// Kill tail when the client disconnects
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
	// tail -f runs indefinitely until killed
}

// BuildNextBinary builds the next binary from a project source with SSE streaming
func BuildNextBinary(w http.ResponseWriter, r *http.Request, projectID string, currentBinPath string) {
	// Find buildable projects
	buildable, err := FindBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the requested project or use the first available
	var project *BuildableProject
	if projectID != "" {
		for i := range buildable {
			if buildable[i].ID == projectID {
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
	dir := filepath.Dir(currentBinPath)
	currentBase, currentVersion := ParseBinVersion(currentBinPath)
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
	cmd := exec.Command("go", "run", "./script/server/build/for-linux-amd64", "-o", destPath)
	cmd.Dir = project.Dir
	cmd.Env = os.Environ()

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
