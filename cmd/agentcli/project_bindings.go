package agentcli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/gitops/git"
)

// projectBinding maps a remote project directory on a server to a local git repo.
type projectBinding struct {
	Server    string `json:"server"`
	RemoteDir string `json:"remote_dir"`
	LocalPath string `json:"local_path"`
}

func resolveProjectLocalDir(server, remoteDir string) string {
	cfg, err := loadConfig()
	if err != nil {
		return ""
	}
	path, ok := findProjectBinding(cfg, server, remoteDir)
	if !ok {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func worktreeBranchName(worktreePath string) string {
	return filepath.Base(worktreePath)
}

func findProjectBinding(cfg *agentConfig, server, remoteDir string) (string, bool) {
	if cfg == nil {
		return "", false
	}
	server = normalizeServerForMatch(server)
	remoteDir = filepath.Clean(remoteDir)
	for _, b := range cfg.ProjectBindings {
		bServer := normalizeServerForMatch(b.Server)
		if bServer != "" && bServer != server {
			continue
		}
		if filepath.Clean(b.RemoteDir) == remoteDir {
			return b.LocalPath, true
		}
	}
	return "", false
}

func upsertProjectBinding(cfg *agentConfig, server, remoteDir, localPath string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	server = normalizeServerForMatch(server)
	remoteDir = filepath.Clean(remoteDir)
	localPath = filepath.Clean(localPath)
	for i := range cfg.ProjectBindings {
		if normalizeServerForMatch(cfg.ProjectBindings[i].Server) == server &&
			filepath.Clean(cfg.ProjectBindings[i].RemoteDir) == remoteDir {
			cfg.ProjectBindings[i].LocalPath = localPath
			cfg.ProjectBindings[i].Server = server
			cfg.ProjectBindings[i].RemoteDir = remoteDir
			return nil
		}
	}
	cfg.ProjectBindings = append(cfg.ProjectBindings, projectBinding{
		Server:    server,
		RemoteDir: remoteDir,
		LocalPath: localPath,
	})
	return nil
}

func localGitOriginURL(dir string) (string, error) {
	ok, err := git.IsInsideGit(dir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%s is not a git repository", dir)
	}
	origin, err := git.GetOriginURL(dir)
	if err != nil {
		return "", fmt.Errorf("read local git origin: %w", err)
	}
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", fmt.Errorf("local repository has no origin remote")
	}
	return origin, nil
}

// sameGitOrigin reports whether two git remote URLs refer to the same repository.
func sameGitOrigin(a, b string) bool {
	na, errA := normalizeGitOriginURL(a)
	nb, errB := normalizeGitOriginURL(b)
	if errA != nil || errB != nil {
		return false
	}
	return na == nb
}

func normalizeGitOriginURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty origin url")
	}
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.TrimSuffix(raw, "/")

	if strings.HasPrefix(raw, "file://") {
		p := strings.TrimPrefix(raw, "file://")
		abs, err := filepath.Abs(filepath.Clean(p))
		if err != nil {
			return "", err
		}
		return "file://" + abs, nil
	}

	if strings.HasPrefix(raw, "git@") {
		at := strings.Index(raw, "@")
		colon := strings.Index(raw, ":")
		if at < 0 || colon <= at {
			return "", fmt.Errorf("invalid scp-style git url %q", raw)
		}
		host := strings.ToLower(raw[at+1 : colon])
		path := strings.TrimPrefix(raw[colon+1:], "/")
		return host + "/" + strings.ToLower(path), nil
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		host := strings.ToLower(u.Hostname())
		path := strings.Trim(strings.TrimSuffix(u.Path, "/"), "/")
		return host + "/" + strings.ToLower(path), nil
	}

	abs, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", err
	}
	return "file://" + abs, nil
}

func serverSlug(server string) string {
	s := normalizeServerForMatch(server)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	replacer := strings.NewReplacer(
		":", "-",
		".", "-",
		"/", "-",
		"_", "-",
	)
	return replacer.Replace(s)
}

func projectSlug(projectName, projectDir string) string {
	name := strings.TrimSpace(projectName)
	if name == "" {
		name = filepath.Base(projectDir)
	}
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
	)
	return replacer.Replace(name)
}

func projectWorktreesRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "remote-agent", "project-worktrees"), nil
}

// allocateWorktreeDir returns .../<branch>-N under server/project slug dirs.
func allocateWorktreeDir(server, projectName, projectDir, branch string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "detached"
	}
	root, err := projectWorktreesRoot()
	if err != nil {
		return "", err
	}
	parent := filepath.Join(root, serverSlug(server), projectSlug(projectName, projectDir))
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}
	suffix, err := nextWorktreeSuffix(parent, branch)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, suffix), nil
}

func nextWorktreeSuffix(parentDir, branch string) (string, error) {
	prefix := branch + "-"
	maxN := 0
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return prefix + "1", nil
		}
		return "", err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(name, prefix+"%d", &n); err == nil && n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("%s%d", prefix, maxN+1), nil
}