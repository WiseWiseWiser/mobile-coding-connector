package agentcli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	localinstall "github.com/xhd2015/dot-pkgs/go-pkgs/gotool/localbin/install"
	"github.com/xhd2015/dot-pkgs/go-pkgs/gotool/mod/installplan"
)

func runInstallBuild(moduleRoot string, item installplan.Item, goos, goarch, stageDir string, stdout, stderr io.Writer, stageN, stageTotal int) error {
	ownRoot, ownRel, err := resolveGoPackageRoot(moduleRoot, item.RelPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return err
	}

	cross := (goos != "" && goos != runtime.GOOS) || (goarch != "" && goarch != runtime.GOARCH)
	env := filterInstallEnv(os.Environ(), cross)

	var argv []string
	switch item.Method {
	case installplan.MethodGoInstall:
		// go install refuses to cross-compile when GOBIN is set; go build -o
		// writes the product into the staging dir for any GOOS/GOARCH.
		if goos != "" {
			env = append(env, "GOOS="+goos)
		}
		if goarch != "" {
			env = append(env, "GOARCH="+goarch)
		}
		if cross {
			env = append(env, "CGO_ENABLED=0")
		}
		outName := item.BinName
		if goos == "windows" && !strings.HasSuffix(strings.ToLower(outName), ".exe") {
			outName += ".exe"
		}
		argv = []string{"go", "build", "-o", filepath.Join(stageDir, outName), ownRel}
	case installplan.MethodGoRunInstall:
		// go run must stay host-native. Pass the target via INSTALL_* so the
		// script can apply GOOS/GOARCH only to the product build.
		if goos != "" {
			env = append(env, localinstall.EnvInstallGOOS+"="+goos)
		}
		if goarch != "" {
			env = append(env, localinstall.EnvInstallGOARCH+"="+goarch)
		}
		env = append(env, localinstall.EnvInstallToDir+"="+stageDir)
		argv = []string{"go", "run", ownRel}
	default:
		return fmt.Errorf("unknown install method %q for %s", item.Method, item.BinName)
	}

	if stageN > 0 && stageTotal > 0 {
		stageDetail(stderr, stageN, stageTotal, "notice: %s", formatInstallBuildNotice(goos, goarch, argv, cross))
	}

	indent := ""
	if stageN > 0 && stageTotal > 0 {
		indent = strings.Repeat(" ", len(fmt.Sprintf("[%d/%d] ", stageN, stageTotal)))
	}
	outW := newIndentingWriter(stdout, indent)
	errW := newIndentingWriter(stderr, indent)
	defer func() {
		_ = outW.Flush()
		_ = errW.Flush()
	}()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = ownRoot
	cmd.Env = env
	cmd.Stdout = outW
	cmd.Stderr = errW
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
	}
	return nil
}

func formatInstallBuildNotice(goos, goarch string, argv []string, cross bool) string {
	envPrefix := ""
	if goos != "" {
		envPrefix += "GOOS=" + goos
	}
	if goarch != "" {
		if envPrefix != "" {
			envPrefix += " "
		}
		envPrefix += "GOARCH=" + goarch
	}
	if cross {
		if envPrefix != "" {
			envPrefix += " "
		}
		envPrefix += "CGO_ENABLED=0"
	}
	cmd := strings.Join(argv, " ")
	if envPrefix == "" {
		return cmd
	}
	return envPrefix + " " + cmd
}

func filterInstallEnv(env []string, cross bool) []string {
	drop := map[string]bool{
		"GOOS":                        true,
		"GOARCH":                      true,
		"GOBIN":                       true,
		"CGO_ENABLED":                 true,
		localinstall.EnvInstallToDir:  true,
		localinstall.EnvInstallGOOS:   true,
		localinstall.EnvInstallGOARCH: true,
	}
	if cross {
		drop["GOFLAGS"] = true
	}
	out := make([]string, 0, len(env)+4)
	for _, e := range env {
		key, _, ok := strings.Cut(e, "=")
		if ok && drop[key] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func collectStagingBinary(stageDir, wantName string) (built string, extras []string, err error) {
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return "", nil, err
	}
	var found string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		p := filepath.Join(stageDir, e.Name())
		if e.Name() == wantName {
			found = p
			continue
		}
		if info.Mode()&0o111 != 0 {
			extras = append(extras, e.Name())
		}
	}
	if found == "" {
		return "", nil, fmt.Errorf("no binary %q in staging dir %s\n  the install script must copy <cmd> into $INSTALL_TO_DIR", wantName, stageDir)
	}
	return found, extras, nil
}

// resolveGoPackageRoot is duplicated from wrk's reinstall execute path:
// walk up from the package dir to the nearest go.mod under moduleRoot.
func resolveGoPackageRoot(moduleRoot, relPath string) (ownRoot, ownRel string, err error) {
	moduleRoot = filepath.Clean(moduleRoot)
	if abs, aerr := filepath.Abs(moduleRoot); aerr == nil {
		moduleRoot = abs
	}
	if resolved, rerr := filepath.EvalSymlinks(moduleRoot); rerr == nil {
		moduleRoot = resolved
	}

	rel := strings.TrimPrefix(relPath, "./")
	pkgDir := filepath.Clean(filepath.Join(moduleRoot, filepath.FromSlash(rel)))
	if resolved, rerr := filepath.EvalSymlinks(pkgDir); rerr == nil {
		pkgDir = resolved
	}

	dir := pkgDir
	for {
		if !installPathUnderOrEqual(dir, moduleRoot) {
			break
		}
		if _, serr := os.Stat(filepath.Join(dir, "go.mod")); serr == nil {
			ownRoot = dir
			relToPkg, rerr := filepath.Rel(ownRoot, pkgDir)
			if rerr != nil {
				return "", "", rerr
			}
			relToPkg = filepath.ToSlash(relToPkg)
			if relToPkg == "" || relToPkg == "." {
				ownRel = "."
			} else {
				ownRel = "./" + relToPkg
			}
			return ownRoot, ownRel, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("no go.mod found for package %s under module root %s", relPath, moduleRoot)
}

func installPathUnderOrEqual(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
