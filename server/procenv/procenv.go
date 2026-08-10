// Package procenv builds environment blocks for managed local subprocesses
// (services, cron) under a thin GUI/server PATH.
package procenv

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/agent-pro/agent/exec/tool_resolve"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/lookpath"
)

// DefaultManagedCLIs are bare tool names ensured via lookpath.LookupPaths
// when missing from the process PATH (best-effort login-shell discovery).
var DefaultManagedCLIs = []string{"go", "node", "npm"}

// BuildManagedEnv returns an env suitable for bash -lc managed jobs:
//  1. process env + tool_resolve extra PATH dirs
//  2. optional ExtraEnv (service/cron overrides; later keys win)
//  3. prepend dirs from LookupPaths(DefaultManagedCLIs) for still-needed tools
//
// Login probe failures are ignored (best-effort). Never mutates os.Environ.
func BuildManagedEnv(extra map[string]string) []string {
	env := tool_resolve.AppendExtraPaths(os.Environ())
	env = mergeExtraEnv(env, extra)
	env = enrichWithLookupPaths(env, DefaultManagedCLIs)
	return env
}

func mergeExtraEnv(env []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return env
	}
	envMap := make(map[string]string, len(env)+len(extra))
	for _, item := range env {
		if idx := strings.IndexByte(item, '='); idx >= 0 {
			envMap[item[:idx]] = item[idx+1:]
		}
	}
	for key, value := range extra {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		envMap[key] = value
	}
	merged := make([]string, 0, len(envMap))
	for key, value := range envMap {
		merged = append(merged, key+"="+value)
	}
	return merged
}

func enrichWithLookupPaths(env []string, names []string) []string {
	if len(names) == 0 {
		return env
	}
	// Only probe names not already found on current PATH.
	need := namesMissingOnPath(env, names)
	if len(need) == 0 {
		return env
	}
	items, err := lookpath.LookupPaths(need, lookpath.Options{})
	if err != nil || len(items) == 0 {
		return env
	}
	dirs := items.Dirs()
	if len(dirs) == 0 {
		return env
	}
	return prependPathDirs(env, dirs)
}

func namesMissingOnPath(env []string, names []string) []string {
	pathVal := lookupEnv(env, "PATH")
	var need []string
	for _, name := range names {
		if name == "" {
			continue
		}
		if findExecutableOnPath(pathVal, name) {
			continue
		}
		need = append(need, name)
	}
	return need
}

func findExecutableOnPath(pathVal, name string) bool {
	if pathVal == "" {
		return false
	}
	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, name)
		if lookpath.IsExecutable(p) {
			return true
		}
	}
	return false
}

func prependPathDirs(env []string, dirs []string) []string {
	if len(dirs) == 0 {
		return env
	}
	pathVal := lookupEnv(env, "PATH")
	existing := make(map[string]struct{})
	for _, d := range filepath.SplitList(pathVal) {
		if d == "" {
			continue
		}
		existing[filepath.Clean(d)] = struct{}{}
	}
	var prefix []string
	for _, d := range dirs {
		d = filepath.Clean(d)
		if d == "" || d == "." {
			continue
		}
		if _, ok := existing[d]; ok {
			continue
		}
		existing[d] = struct{}{}
		prefix = append(prefix, d)
	}
	if len(prefix) == 0 {
		return env
	}
	newPath := strings.Join(prefix, string(os.PathListSeparator))
	if pathVal != "" {
		newPath = newPath + string(os.PathListSeparator) + pathVal
	}
	return setEnv(env, "PATH", newPath)
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	found := false
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			out = append(out, prefix+value)
			found = true
			continue
		}
		out = append(out, item)
	}
	if !found {
		out = append(out, prefix+value)
	}
	return out
}
