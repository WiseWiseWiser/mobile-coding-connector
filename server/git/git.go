// Package git exposes HTTP endpoints for git operations that run on the
// server host. It currently offers:
//
//	POST /api/remote-agent/git/clone  — clone a repository into
//	                                    ~/<repo_base_name> or a caller-
//	                                    supplied directory.
//	POST /api/remote-agent/git/fetch  — git fetch inside an existing
//	                                    repository.
//	POST /api/remote-agent/git/pull   — git pull --ff-only inside an
//	                                    existing repository.
//
// These paths are namespaced under /api/remote-agent/ to avoid colliding
// with /api/git/{fetch,pull,push} owned by server/github, which accepts
// a different request shape (project_id + encrypted SSH key).
//
// Every endpoint streams stdout/stderr back to the client as
// newline-delimited JSON (NDJSON) events, using the same event protocol
// as /api/exec:
//
//	{"type":"stdout","data":"..."}
//	{"type":"stderr","data":"..."}
//	{"type":"heartbeat"}
//	{"type":"exit","code":N}
//	{"type":"error","message":"..."}
package git

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/gitutil"
	"github.com/xhd2015/lifelog-private/ai-critic/server/ndjsonstream"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/proxyselect"
)

// CloneRequest is the JSON body accepted by POST /api/remote-agent/git/clone.
type CloneRequest struct {
	// Repo is the repository URL to clone. Required.
	Repo string `json:"repo"`
	// Dir is the target directory on the server. If empty, the server
	// clones into ~/<repo_base_name>.
	Dir string `json:"dir"`
	// PrivateKey is the raw contents of an SSH private key. If non-empty,
	// the server writes it to a temporary file, points GIT_SSH_COMMAND at
	// it for the clone, and removes the file when the clone finishes.
	PrivateKey string `json:"private_key"`
	// HTTPSProxy is the value to export as https_proxy / HTTPS_PROXY for
	// the git process. Optional.
	HTTPSProxy string `json:"https_proxy"`
	// SSHUser is the user component used when the server rewrites an
	// HTTPS URL to its SSH form (see ToSSH). Only consulted when a
	// PrivateKey is also provided. Defaults to "git" when empty.
	SSHUser string `json:"ssh_user"`
}

// RepoOpRequest is the JSON body accepted by POST /api/remote-agent/git/fetch
// and POST /api/remote-agent/git/pull. Dir is required and must be an
// existing git repository.
type RepoOpRequest struct {
	Dir        string `json:"dir"`
	PrivateKey string `json:"private_key"`
	HTTPSProxy string `json:"https_proxy"`
}

// RegisterAPI registers the /api/remote-agent/git/* endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/remote-agent/git/clone", handleClone)
	mux.HandleFunc("/api/remote-agent/git/fetch", handleFetch)
	mux.HandleFunc("/api/remote-agent/git/pull", handlePull)
}

func handleClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req CloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.Repo == "" {
		writeJSONError(w, http.StatusBadRequest, "repo is required")
		return
	}
	if err := gitrunner.EnsureAvailable(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	targetDir, err := resolveCloneTargetDir(req.Repo, req.Dir)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, statErr := os.Stat(targetDir); statErr == nil {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("target already exists: %s", targetDir))
		return
	} else if !os.IsNotExist(statErr) {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("stat target: %v", statErr))
		return
	}

	// If the caller supplied an SSH private key but an HTTPS repo URL,
	// rewrite the URL to its SSH form so the key is actually used.
	repo := req.Repo
	var rewriteNote string
	if req.PrivateKey != "" && !gitutil.IsSSH(repo) {
		sshURL := gitutil.ToSSH(repo, req.SSHUser)
		if sshURL != repo {
			rewriteNote = fmt.Sprintf("rewrote HTTPS repo URL to SSH (using key): %s -> %s", repo, sshURL)
			repo = sshURL
		}
	}

	proxy := proxyselect.ForRepoURL(req.HTTPSProxy, repo)

	note := joinNotes(rewriteNote, proxy.Note)

	runStreaming(w, r, req.PrivateKey, note, func(keyPath string) *exec.Cmd {
		gc := gitrunner.Clone(repo, targetDir)
		return applyCommonOpts(gc, keyPath, proxy.URL).Exec()
	})
}

// joinNotes concatenates non-empty notes with newlines so multiple
// informational preambles can be emitted as a single preamble block.
func joinNotes(notes ...string) string {
	var parts []string
	for _, n := range notes {
		if n != "" {
			parts = append(parts, n)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

func handleFetch(w http.ResponseWriter, r *http.Request) {
	handleRepoOp(w, r, func(dir string, req RepoOpRequest) (string, func(keyPath string) *exec.Cmd) {
		proxy := proxyselect.ForRepoDir(req.HTTPSProxy, dir)
		return proxy.Note, func(keyPath string) *exec.Cmd {
			gc := gitrunner.Fetch().Dir(dir)
			return applyCommonOpts(gc, keyPath, proxy.URL).Exec()
		}
	})
}

func handlePull(w http.ResponseWriter, r *http.Request) {
	handleRepoOp(w, r, func(dir string, req RepoOpRequest) (string, func(keyPath string) *exec.Cmd) {
		proxy := proxyselect.ForRepoDir(req.HTTPSProxy, dir)
		return proxy.Note, func(keyPath string) *exec.Cmd {
			gc := gitrunner.PullFFOnly().Dir(dir)
			return applyCommonOpts(gc, keyPath, proxy.URL).Exec()
		}
	})
}

// handleRepoOp decodes a RepoOpRequest, validates that Dir is an existing
// git repository, and streams the command built by makeCmdBuilder back to
// the client. makeCmdBuilder returns an optional human-readable note that
// is emitted on the stream before the command starts (used e.g. to
// announce an auto-selected proxy), along with the actual command
// factory.
func handleRepoOp(w http.ResponseWriter, r *http.Request, makeCmdBuilder func(dir string, req RepoOpRequest) (note string, makeCmd func(keyPath string) *exec.Cmd)) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req RepoOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.Dir == "" {
		writeJSONError(w, http.StatusBadRequest, "dir is required")
		return
	}
	if err := gitrunner.EnsureAvailable(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	dir, err := absPath(req.Dir)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, statErr := os.Stat(dir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("dir does not exist: %s", dir))
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("stat dir: %v", statErr))
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir is not a directory: %s", dir))
		return
	}
	if !gitrunner.IsRepo(dir) {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir is not a git repository: %s", dir))
		return
	}

	note, makeCmd := makeCmdBuilder(dir, req)
	runStreaming(w, r, req.PrivateKey, note, makeCmd)
}

// runStreaming materializes the optional private key, opens the NDJSON
// response stream, optionally emits a preamble note (on stderr), starts
// the command produced by makeCmd, pumps its stdout/stderr through the
// stream, and emits the final exit event.
//
// The command process is killed when the HTTP client disconnects. All
// heartbeat and cleanup plumbing is handled here so individual handlers
// stay short.
func runStreaming(w http.ResponseWriter, r *http.Request, privateKey string, note string, makeCmd func(keyPath string) *exec.Cmd) {
	keyPath, cleanupKey, err := writePrivateKey(privateKey)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("write private key: %v", err))
		return
	}
	defer cleanupKey()

	// Switch response to NDJSON streaming BEFORE we start writing events.
	stream := ndjsonstream.NewWriter(w)

	// Emit an informational note (e.g. "auto-selected proxy ...") before
	// the command's own output so the client sees it in context. The
	// note is written to the stderr channel so it visually groups with
	// git's own progress output.
	if note != "" {
		stream.Send(map[string]any{"type": "stderr", "data": note + "\n"})
	}

	stopHeartbeat := make(chan struct{})
	var heartbeatDone sync.WaitGroup
	heartbeatDone.Add(1)
	go func() {
		defer heartbeatDone.Done()
		ndjsonstream.RunHeartbeat(stream, ndjsonstream.HeartbeatInterval, stopHeartbeat)
	}()
	defer func() {
		close(stopHeartbeat)
		heartbeatDone.Wait()
	}()

	cmd := makeCmd(keyPath)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stream.SendError(fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stream.SendError(fmt.Sprintf("stderr pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		stream.SendError(fmt.Sprintf("failed to start git: %v", err))
		return
	}

	// Kill the git process if the client disconnects before it finishes,
	// so we don't leave orphaned work (possibly writing large amounts of
	// data to disk).
	ctxDone := r.Context().Done()
	cancelled := make(chan struct{})
	go func() {
		select {
		case <-ctxDone:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		case <-cancelled:
		}
	}()
	defer close(cancelled)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pumpPipe(stdoutPipe, "stdout", stream)
	}()
	go func() {
		defer wg.Done()
		pumpPipe(stderrPipe, "stderr", stream)
	}()
	wg.Wait()

	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			stream.SendError(fmt.Sprintf("wait: %v", waitErr))
			return
		}
	}

	stream.Send(map[string]any{"type": "exit", "code": exitCode})
}

// applyCommonOpts wires the optional SSH key and proxy settings onto a
// gitrunner command. Both knobs are shared by clone/fetch/pull.
func applyCommonOpts(gc *gitrunner.Command, keyPath string, httpsProxy string) *gitrunner.Command {
	if keyPath != "" || httpsProxy != "" {
		gc = gc.WithSSHConfig(&gitrunner.SSHKeyConfig{
			KeyPath:  keyPath,
			ProxyURL: httpsProxy,
		})
	}
	if httpsProxy != "" {
		gc = gc.WithEnv("https_proxy", httpsProxy).WithEnv("HTTPS_PROXY", httpsProxy)
	}
	return gc
}

// resolveCloneTargetDir returns dir unchanged (as an absolute path) when
// non-empty, or ~/<base> derived from repo otherwise.
func resolveCloneTargetDir(repo string, dir string) (string, error) {
	if dir != "" {
		return absPath(dir)
	}
	base := repoBaseName(repo)
	if base == "" {
		return "", fmt.Errorf("could not derive a directory name from repo %q; pass dir explicitly", repo)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, base), nil
}

// absPath converts a possibly-relative path to an absolute one.
func absPath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve dir %q: %w", p, err)
	}
	return abs, nil
}

// repoBaseName returns the last path segment of repo with a trailing
// '.git' stripped. It handles the three common repo URL forms:
//   - https://host/owner/name(.git)
//   - ssh://git@host/owner/name(.git)
//   - git@host:owner/name(.git)
func repoBaseName(repo string) string {
	s := repo
	if !strings.Contains(s, "://") {
		if idx := strings.LastIndex(s, ":"); idx >= 0 && !strings.Contains(s[:idx], "/") {
			s = s[idx+1:]
		}
	}
	s = strings.TrimRight(s, "/")
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimSuffix(s, ".git")
	return s
}

// writePrivateKey materializes the given key contents into a temporary
// file with 0600 permissions. Returns the path and a cleanup function
// that removes the file; cleanup is always safe to call. When contents
// is empty, the returned path is "" and cleanup is a no-op.
func writePrivateKey(contents string) (string, func(), error) {
	if contents == "" {
		return "", func() {}, nil
	}

	// Keep the key outside of the repo-clone area so a stray rm -rf of
	// the target directory can't take it with it. Use a per-invocation
	// suffix so concurrent operations don't collide.
	dir := filepath.Join(os.TempDir(), "remote-agent-git")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", func() {}, err
	}
	suffix, err := randomSuffix()
	if err != nil {
		return "", func() {}, err
	}
	path := filepath.Join(dir, "op-key-"+suffix)

	// SSH refuses to use keys that aren't 0600; also, an SSH key must end
	// with a newline to be parsed correctly by older ssh versions.
	data := contents
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		return "", func() {}, err
	}

	cleanup := func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

func randomSuffix() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func pumpPipe(pipe io.Reader, kind string, stream *ndjsonstream.Writer) {
	buf := make([]byte, 32*1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			stream.Send(map[string]any{"type": kind, "data": safeString(buf[:n])})
		}
		if err != nil {
			return
		}
	}
}

func safeString(b []byte) string {
	return strings.ToValidUTF8(string(b), "\uFFFD")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
