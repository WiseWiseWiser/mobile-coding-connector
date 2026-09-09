package agentcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/dot-pkgs/go-pkgs/git/scan_repo"
	"github.com/xhd2015/less-gen/flags"
)

// Canonical git origin for ai-critic source checkouts (module path differs).
const aiCriticCanonicalOrigin = "https://github.com/WiseWiseWiser/mobile-coding-connector.git"

const serverUpgradeHelp = `Usage: remote-agent server upgrade --from-source [--source-dir DIR] [options]

Build the ai-critic server from local source for the remote OS/arch, upload it
via upload-next, and restart the remote server.

When --source-dir is omitted, scan local git checkouts whose origin matches
ai-critic's canonical origin (github.com/WiseWiseWiser/mobile-coding-connector),
preferring a checkout under the current directory, then one that contains cwd,
then a main checkout. Prints which source directory is used.

Options:
  --from-source         Required. Build from local source (this mode).
  --source-dir DIR      Explicit ai-critic source checkout (skips origin scan).
  --goos OS             Override target GOOS (default: from remote os_info / binary name).
  --goarch ARCH         Override target GOARCH (default: from remote os_info / binary name).
  --skip-frontend       Skip the Vite frontend build.
  --dry-run             Resolve source and target, print steps, do not build/upload/restart.
  -h, --help            Show this help message.
`

func runServerUpgrade(resolve func() (*client.Client, error), args []string) error {
	var fromSource bool
	var sourceDir string
	var goosFlag string
	var goarchFlag string
	var skipFrontend bool
	var dryRun bool

	args, err := flags.
		Bool("--from-source", &fromSource).
		String("--source-dir", &sourceDir).
		String("--goos", &goosFlag).
		String("--goarch", &goarchFlag).
		Bool("--skip-frontend", &skipFrontend).
		Bool("--dry-run", &dryRun).
		Help("-h,--help", serverUpgradeHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("server upgrade does not accept positional args: %v", args)
	}
	if !fromSource {
		return fmt.Errorf("server upgrade requires --from-source (see --help)")
	}

	resolvedDir, notice, err := resolveUpgradeSourceDir(sourceDir)
	if err != nil {
		return err
	}
	fmt.Println(notice)

	if err := validateAICriticSourceDir(resolvedDir); err != nil {
		return err
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	goos, goarch, targetSrc, err := resolveUpgradeTarget(cli, goosFlag, goarchFlag)
	if err != nil {
		return err
	}
	fmt.Printf("Target: %s/%s (%s)\n", goos, goarch, targetSrc)

	outName := fmt.Sprintf("ai-critic-server-%s-%s", goos, goarch)
	outPath := filepath.Join(resolvedDir, outName)

	if dryRun {
		fmt.Println("dry-run: would build frontend (unless --skip-frontend)")
		fmt.Printf("dry-run: would cross-compile -> %s\n", outPath)
		fmt.Println("dry-run: would upload-next and restart")
		return nil
	}

	if err := buildUpgradeBinary(resolvedDir, outPath, goos, goarch, skipFrontend); err != nil {
		return err
	}

	if err := uploadLocalBinaryNext(cli, outPath); err != nil {
		return err
	}

	result, err := cli.RestartServer(func(ev client.ServerStreamEvent) {
		if ev.Message != "" {
			fmt.Println(ev.Message)
		}
	})
	if err != nil {
		return err
	}
	printRestartResult(result)
	return nil
}

func resolveUpgradeSourceDir(explicit string) (dir string, notice string, err error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		abs, absErr := filepath.Abs(explicit)
		if absErr != nil {
			return "", "", fmt.Errorf("resolve --source-dir: %w", absErr)
		}
		return abs, fmt.Sprintf("using source: %s (--source-dir)", abs), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("getwd: %w", err)
	}

	// Prefer a fast scan of cwd's git toplevel (common when already in/near source),
	// then fall back to $HOME + toplevel via DefaultResolveRoots (cached Scan).
	var resolved *scan_repo.ResolveByOriginResult
	if roots, rootErr := scan_repo.DefaultResolveRoots(cwd); rootErr == nil && len(roots) > 1 {
		// roots[1] is cwd toplevel when distinct from $HOME.
		fast := scan_repo.Options{Roots: []string{roots[len(roots)-1]}}
		resolved, err = scan_repo.ResolveByOrigin(context.Background(), fast, aiCriticCanonicalOrigin, cwd)
	}
	if resolved == nil {
		roots, rootErr := scan_repo.DefaultResolveRoots(cwd)
		if rootErr != nil {
			return "", "", rootErr
		}
		resolved, err = scan_repo.ResolveByOrigin(context.Background(), scan_repo.Options{
			Roots: roots,
		}, aiCriticCanonicalOrigin, cwd)
	}
	if err != nil {
		return "", "", fmt.Errorf("%w\nhint: pass --source-dir, or clone/bring ai-critic under a scan root", err)
	}
	return resolved.Path, fmt.Sprintf("using source: %s (%s)", resolved.Path, resolved.Reason), nil
}

func validateAICriticSourceDir(dir string) error {
	goMod := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("source dir %s does not look like ai-critic (read go.mod: %w)", dir, err)
	}
	if !strings.Contains(string(data), "module github.com/xhd2015/ai-critic") {
		return fmt.Errorf("source dir %s go.mod is not module github.com/xhd2015/ai-critic", dir)
	}
	bundle := filepath.Join(dir, "script", "bundle", "for-linux")
	if _, err := os.Stat(bundle); err != nil {
		return fmt.Errorf("source dir %s missing script/bundle/for-linux: %w", dir, err)
	}
	return nil
}

func resolveUpgradeTarget(cli *client.Client, goosFlag, goarchFlag string) (goos, goarch, source string, err error) {
	goosFlag = strings.TrimSpace(goosFlag)
	goarchFlag = strings.TrimSpace(goarchFlag)
	if goosFlag != "" && goarchFlag != "" {
		return strings.ToLower(goosFlag), strings.ToLower(goarchFlag), "flags", nil
	}

	var fromBinaryGOOS, fromBinaryGOARCH string
	if status, statusErr := cli.GetKeepAliveStatus(); statusErr == nil && status != nil {
		fromBinaryGOOS, fromBinaryGOARCH, _ = parseGOOSArchFromBinaryName(filepath.Base(status.BinaryPath))
	}

	serverStatus, statusErr := cli.GetServerStatus()
	var fromInfoGOOS, fromInfoGOARCH string
	if statusErr == nil && serverStatus != nil {
		fromInfoGOOS = mapUnameToGOOS(serverStatus.OSInfo.OS)
		fromInfoGOARCH = mapUnameToGOARCH(serverStatus.OSInfo.Arch)
	}

	parts := make([]string, 0, 3)

	goos = strings.ToLower(goosFlag)
	if goos == "" {
		if fromBinaryGOOS != "" {
			goos = fromBinaryGOOS
			parts = append(parts, "binary name")
		} else if fromInfoGOOS != "" {
			goos = fromInfoGOOS
			parts = append(parts, "remote os_info")
		} else {
			goos = "linux"
			parts = append(parts, "default")
		}
	} else {
		parts = append(parts, "--goos")
	}

	goarch = strings.ToLower(goarchFlag)
	if goarch == "" {
		if fromBinaryGOARCH != "" {
			goarch = fromBinaryGOARCH
			if !containsString(parts, "binary name") {
				parts = append(parts, "binary name")
			}
		} else if fromInfoGOARCH != "" {
			goarch = fromInfoGOARCH
			if !containsString(parts, "remote os_info") {
				parts = append(parts, "remote os_info")
			}
		} else {
			goarch = "amd64"
			if !containsString(parts, "default") {
				parts = append(parts, "default")
			}
		}
	} else {
		parts = append(parts, "--goarch")
	}

	return goos, goarch, strings.Join(parts, "+"), nil
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func mapUnameToGOOS(osInfo string) string {
	s := strings.ToLower(osInfo)
	switch {
	case strings.Contains(s, "darwin"), strings.Contains(s, "macos"):
		return "darwin"
	case strings.Contains(s, "windows"):
		return "windows"
	case s == "":
		return ""
	default:
		// Ubuntu, GNU/Linux, Linux, …
		return "linux"
	}
}

func mapUnameToGOARCH(unameM string) string {
	switch strings.ToLower(strings.TrimSpace(unameM)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "i386", "i686", "x86":
		return "386"
	case "":
		return ""
	default:
		return strings.ToLower(strings.TrimSpace(unameM))
	}
}

func parseGOOSArchFromBinaryName(name string) (goos, goarch string, ok bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false
	}
	base, _ := parseRemoteBinaryVersion(name)
	// Expect …-<goos>-<goarch>
	parts := strings.Split(base, "-")
	if len(parts) < 2 {
		return "", "", false
	}
	goarch = parts[len(parts)-1]
	goos = parts[len(parts)-2]
	switch goos {
	case "linux", "darwin", "windows", "freebsd":
	default:
		return "", "", false
	}
	switch goarch {
	case "amd64", "arm64", "386", "arm":
	default:
		return "", "", false
	}
	return goos, goarch, true
}

func buildUpgradeBinary(sourceDir, outPath, goos, goarch string, skipFrontend bool) error {
	prev, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(sourceDir); err != nil {
		return fmt.Errorf("chdir source: %w", err)
	}
	defer func() { _ = os.Chdir(prev) }()

	if !skipFrontend {
		if err := lib.EnsureNodeModules("ai-critic-react"); err != nil {
			return err
		}
		if err := lib.BuildFrontend(); err != nil {
			return err
		}
	}

	return lib.BuildServer(lib.BuildServerOptions{
		Output: outPath,
		GOOS:   goos,
		GOARCH: goarch,
	})
}

func uploadLocalBinaryNext(cli *client.Client, localBinary string) error {
	stat, err := os.Stat(localBinary)
	if err != nil {
		return fmt.Errorf("failed to stat local binary: %w", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("local binary is a directory, not a file: %s", localBinary)
	}

	target, usedCompatTarget, err := getNextBinaryTargetForUpload(cli)
	if err != nil {
		return err
	}
	if strings.TrimSpace(target.BinaryPath) == "" {
		return fmt.Errorf("server returned empty next binary path")
	}
	if usedCompatTarget {
		fmt.Println("Derived target from keep-alive status and remote directory listing.")
	}
	fmt.Printf("Next remote binary: %s\n", target.BinaryPath)
	if target.CurrentPath != "" {
		fmt.Printf("Current remote binary: %s\n", target.CurrentPath)
	}
	if target.PreviousHighestVersion > 0 || target.Version > 0 {
		fmt.Printf("Version: v%d -> v%d\n", target.PreviousHighestVersion, target.Version)
	}
	fmt.Printf("Uploading %s (%s) -> %s\n", localBinary, formatSize(stat.Size()), target.BinaryPath)

	result, err := cli.UploadFile(localBinary, target.BinaryPath, client.UploadOptions{
		ChmodExec: true,
	}, printUploadProgress)
	if err != nil {
		return err
	}
	fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	return nil
}

func printRestartResult(result *client.RestartServerResult) {
	if result == nil {
		fmt.Println("Server is back up and reachable.")
		return
	}
	if result.KeepAlive != nil {
		fmt.Printf("Server is back up: PID %d  Uptime %s  Restarts %d\n",
			result.KeepAlive.ServerPID,
			displayOrDash(result.KeepAlive.Uptime),
			result.KeepAlive.RestartCount,
		)
		fmt.Printf("Binary: %s\n", displayOrDash(result.KeepAlive.BinaryPath))
		return
	}
	if result.Binary != "" {
		fmt.Printf("Server is back up. Requested binary: %s\n", result.Binary)
		return
	}
	fmt.Println("Server is back up and reachable.")
}
