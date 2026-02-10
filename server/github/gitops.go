package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/projects"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// registerGitOpsAPI registers git operation endpoints.
func registerGitOpsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/git/fetch", handleGitFetch)
	mux.HandleFunc("/api/git/pull", handleGitPull)
	mux.HandleFunc("/api/git/push", handleGitPush)
}

type gitOpRequest struct {
	ProjectID string `json:"project_id"`
	// Optional: encrypted SSH key from browser.
	// If empty, uses the project's configured SSH key (looked up from browser storage via ssh_key_id).
	SSHKey string `json:"ssh_key,omitempty"`
}

func handleGitFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runGitOp(w, r, "fetch", "--progress")
}

func handleGitPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runGitOp(w, r, "pull", "--progress")
}

func handleGitPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runGitOp(w, r, "push", "--progress")
}

// runGitOp executes a git command (fetch/pull) in the project directory with SSE streaming.
func runGitOp(w http.ResponseWriter, r *http.Request, gitCmd string, gitArgs ...string) {
	var req gitOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	// Look up project
	projectList, err := projects.List()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list projects: %v", err), http.StatusInternalServerError)
		return
	}

	var project *projects.Project
	for i := range projectList {
		if projectList[i].ID == req.ProjectID {
			project = &projectList[i]
			break
		}
	}
	if project == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	// Verify directory exists
	if _, err := os.Stat(project.Dir); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("project directory does not exist: %s", project.Dir), http.StatusBadRequest)
		return
	}

	// Set up SSE streaming
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Prepare SSH key if project uses SSH and encrypted key is provided
	var keyFile *SSHKeyFile
	if req.SSHKey != "" {
		var err error
		keyFile, err = PrepareSSHKeyFile(req.SSHKey)
		if err != nil {
			sw.SendError(fmt.Sprintf("Failed to prepare SSH key: %v", err))
			return
		}
		defer keyFile.Cleanup()
	}

	// Build git command using gitrunner to ensure proper environment
	var keyPath string
	if keyFile != nil {
		keyPath = keyFile.Path
	}

	var cmd *exec.Cmd
	if gitCmd == "push" {
		// For push, get current branch and use explicit upstream format
		branch, err := gitrunner.GetCurrentBranch(project.Dir)
		if err != nil {
			sw.SendError(fmt.Sprintf("Failed to get current branch: %v", err))
			return
		}
		cmd = gitrunner.Push(branch, keyPath).Dir(project.Dir).Exec()
		sw.SendLog(fmt.Sprintf("$ git push origin HEAD:%s %s", branch, project.Dir))
	} else {
		cmd = gitrunner.NewCommand(append([]string{gitCmd}, gitArgs...)...).Dir(project.Dir).WithSSHKey(keyPath).Exec()
		sw.SendLog(fmt.Sprintf("$ git %s %s", gitCmd, project.Dir))
	}

	if keyFile != nil {
		sw.SendLog(fmt.Sprintf("Using SSH key: %s", keyFile.KeyType))
	}

	cmdErr := sw.StreamCmd(cmd)
	if cmdErr != nil {
		sw.SendError(fmt.Sprintf("git %s failed: %v", gitCmd, cmdErr))
		return
	}

	sw.SendDone(map[string]string{
		"message": fmt.Sprintf("git %s completed successfully", gitCmd),
	})
}
