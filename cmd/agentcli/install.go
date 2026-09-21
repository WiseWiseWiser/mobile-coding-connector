package agentcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/dot-pkgs/go-pkgs/file/tmpdir"
	"github.com/xhd2015/dot-pkgs/go-pkgs/gotool/mod/installplan"
	"github.com/xhd2015/less-gen/flags"
)

func installHelp() string {
	name := active.Name
	if name == "" {
		name = "remote-agent"
	}
	return fmt.Sprintf(`Usage: %s install <cmd> [--dir DIR] [options]

Cross-build <cmd> from local Go source for the remote OS/arch, then upload
it to the remote's existing path, or ~/.local/bin if it is not installed yet.
Ensures ~/.local/bin is on PATH in remote shell profiles when installing there.

Build detection matches wrk --reinstall-local:
  ./script/<cmd>/install  >  ./script/install (only when cmd is the module basename)
                          >  ./cmd/<cmd>

Install scripts (go run ./script/.../install) run host-native. The product
build must honor these env vars:

  INSTALL_TO_DIR    directory to write <cmd> (empty after success is an error)
  INSTALL_GOOS      target GOOS for the product binary
  INSTALL_GOARCH    target GOARCH for the product binary

Arguments:
  cmd                 Binary name to install (required)

Options:
  --dir DIR           Source tree (default: current directory; scanned like wrk)
  --goos OS           Override remote GOOS
  --goarch ARCH       Override remote GOARCH
  --dry-run           Probe and print the plan; do not build or upload
  -h, --help          Show this help
`, name)
}

func runInstall(resolve func() (*client.Client, error), args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	var sourceDir string
	var goosFlag string
	var goarchFlag string
	var dryRun bool

	args, err := flags.
		String("--dir", &sourceDir).
		String("--goos", &goosFlag).
		String("--goarch", &goarchFlag).
		Bool("--dry-run", &dryRun).
		HelpFunc("-h,--help", func() {
			fmt.Fprint(stdout, strings.TrimRight(installHelp(), "\n")+"\n")
		}).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("install requires <cmd>; see '%s install --help'", active.Name)
	}
	if len(args) > 1 {
		return fmt.Errorf("install takes one <cmd>, got %d", len(args))
	}
	cmdName := strings.TrimSpace(args[0])
	if err := validateInstallCmdName(cmdName); err != nil {
		return err
	}

	const stages = 4
	stagePrint(stderr, 1, stages, "discover", cmdName)

	workDir := strings.TrimSpace(sourceDir)
	if workDir == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("getwd: %w", cwdErr)
		}
		workDir = cwd
	} else {
		abs, absErr := filepath.Abs(workDir)
		if absErr != nil {
			return fmt.Errorf("resolve --dir: %w", absErr)
		}
		workDir = abs
	}

	stageDetail(stderr, 1, stages, "notice: scanning cmd/ and script/ under %s", workDir)
	plan, err := installplan.DiscoverFromWorkDir(workDir, false)
	if err != nil {
		return err
	}
	plan, err = installplan.Lookup(plan, []string{cmdName})
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	mod := plan.Modules[0]
	item := mod.Items[0]
	stageDetail(stderr, 1, stages, "notice: %s (%s)", item.RelPath, item.Method)
	printInstallDiags(stderr, mod.Diagnostics)

	cli, err := resolve()
	if err != nil {
		return err
	}
	remote := liveInstallRemote{cli: cli}

	goos, goarch, src, warn := resolveInstallTarget(cli, goosFlag, goarchFlag)
	if warn != "" {
		stageDetail(stderr, 1, stages, "warning: %s", warn)
	}
	stageDetail(stderr, 1, stages, "notice: target %s/%s (%s)", goos, goarch, src)

	home, err := remote.Home()
	if err != nil {
		return err
	}
	lookPath, err := remote.LookCommand(cmdName)
	if err != nil {
		return err
	}
	gobin, _ := remote.GoEnv("GOBIN")
	gopath, _ := remote.GoEnv("GOPATH")

	wantName := installBinFileName(cmdName, goos)
	dest := planRemoteInstallDest(wantName, home, lookPath, gobin, gopath)
	dest, err = filterExistingExtras(dest, remote.CheckExists)
	if err != nil {
		return err
	}
	if dest.SystemWarning != "" {
		stageDetail(stderr, 1, 4, "warning: %s", dest.SystemWarning)
	}

	if dryRun {
		return printInstallDryRun(stdout, stderr, item, goos, goarch, dest, remote, home)
	}

	stageDir, err := os.MkdirTemp(tmpdir.GetCommonTmpDir(), "remote-agent-install-*")
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	stagePrint(stderr, 2, stages, "build", fmt.Sprintf("%s/%s → %s", goos, goarch, filepath.Join(stageDir, wantName)))
	if err := runInstallBuild(mod.ModuleRoot, item, goos, goarch, stageDir, stdout, stderr, 2, stages); err != nil {
		return err
	}
	built, extras, err := collectStagingBinary(stageDir, wantName)
	if err != nil {
		return fmt.Errorf("%s: %w", item.RelPath, err)
	}
	for _, extra := range extras {
		stageDetail(stderr, 2, 4, "warning: ignoring extra staging file %s", extra)
	}

	stagePrint(stderr, 3, stages, "upload", fmt.Sprintf("%s → %s", built, dest.Primary))
	if err := remote.UploadExec(built, dest.Primary); err != nil {
		return err
	}
	for _, extra := range dest.Extras {
		stageDetail(stderr, 3, 4, "also %s", extra)
		if err := remote.UploadExec(built, extra); err != nil {
			return fmt.Errorf("upload extra %s: %w", extra, err)
		}
	}

	wroteLocal := destWritesLocalBin(dest, home)
	if wroteLocal {
		stagePrint(stderr, 4, stages, "path", "ensure ~/.local/bin on PATH")
		if err := ensureRemoteLocalBinPATH(remote, home, false, stdout, stderr); err != nil {
			return err
		}
	} else {
		stagePrint(stderr, 4, stages, "path", "skipped (not writing ~/.local/bin)")
	}

	fmt.Fprintln(stderr)
	fmt.Fprintf(stdout, "installed %s → %s (%s/%s)\n", cmdName, dest.Primary, goos, goarch)
	return nil
}

const installTargetTimeout = 2 * time.Second

// resolveInstallTarget picks GOOS/GOARCH without waiting on full /api/server/status
// (df/ps). Soft-fails to linux/amd64 after installTargetTimeout.
func resolveInstallTarget(cli *client.Client, goosFlag, goarchFlag string) (goos, goarch, source, warn string) {
	goosFlag = strings.TrimSpace(goosFlag)
	goarchFlag = strings.TrimSpace(goarchFlag)
	if goosFlag != "" && goarchFlag != "" {
		return strings.ToLower(goosFlag), strings.ToLower(goarchFlag), "flags", ""
	}
	goos = strings.ToLower(goosFlag)
	goarch = strings.ToLower(goarchFlag)
	var parts []string

	ctx, cancel := context.WithTimeout(context.Background(), installTargetTimeout)
	defer cancel()
	if status, err := cli.GetKeepAliveStatusContext(ctx); err == nil && status != nil {
		bg, ba, _ := parseGOOSArchFromBinaryName(filepath.Base(status.BinaryPath))
		if goos == "" && bg != "" {
			goos = bg
			parts = append(parts, "binary name")
		}
		if goarch == "" && ba != "" {
			goarch = ba
			if !containsString(parts, "binary name") {
				parts = append(parts, "binary name")
			}
		}
	}

	if goos == "" || goarch == "" {
		osCtx, osCancel := context.WithTimeout(context.Background(), installTargetTimeout)
		defer osCancel()
		if info, err := cli.GetOSInfo(osCtx); err == nil && info != nil {
			if goos == "" {
				if mapped := mapUnameToGOOS(info.OS); mapped != "" {
					goos = mapped
					parts = append(parts, "remote os-info")
				}
			}
			if goarch == "" {
				if mapped := mapUnameToGOARCH(info.Arch); mapped != "" {
					goarch = mapped
					if !containsString(parts, "remote os-info") {
						parts = append(parts, "remote os-info")
					}
				}
			}
		} else if err != nil && warn == "" {
			msg := err.Error()
			if strings.Contains(msg, "invalid character '<'") || strings.Contains(msg, "404") {
				warn = "remote has no /api/server/os-info yet; using linux/amd64 if unset"
			} else {
				warn = fmt.Sprintf("remote os_info failed (%v); using linux/amd64 if unset", err)
			}
		}
	}

	if goos == "" {
		goos = "linux"
		parts = append(parts, "default")
	}
	if goarch == "" {
		goarch = "amd64"
		if !containsString(parts, "default") {
			parts = append(parts, "default")
		}
	}
	source = strings.Join(parts, "+")
	if source == "" {
		source = "default"
	}
	return goos, goarch, source, warn
}

func validateInstallCmdName(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid command name %q", name)
	}
	if strings.ContainsAny(name, "/\\ \t\n$;&|<>()") {
		return fmt.Errorf("invalid command name %q", name)
	}
	return nil
}

func installBinFileName(cmd, goos string) string {
	if goos == "windows" && !strings.HasSuffix(strings.ToLower(cmd), ".exe") {
		return cmd + ".exe"
	}
	return cmd
}

func printInstallDiags(stderr io.Writer, diags []installplan.Diagnostic) {
	for _, d := range diags {
		switch d.Kind {
		case installplan.DiagKindPreferScript:
			if len(d.Paths) == 0 {
				continue
			}
			winner, rest := d.Paths[0], d.Paths[1:]
			stageDetail(stderr, 1, 4, "notice: preferring %s over %s", winner, strings.Join(rest, ", "))
		case installplan.DiagKindNestedScript:
			if len(d.Paths) == 0 {
				continue
			}
			winner, rest := d.Paths[0], d.Paths[1:]
			stageDetail(stderr, 1, 4, "warning: ignoring nested script (%s); using %s", strings.Join(rest, ", "), winner)
		case installplan.DiagKindAmbiguousCmd, installplan.DiagKindAmbiguousScript:
			stageDetail(stderr, 1, 4, "warning: %s %s", d.Kind, strings.Join(d.Paths, ", "))
		default:
			stageDetail(stderr, 1, 4, "warning: %s", d.Kind)
		}
	}
}

func stagePrint(w io.Writer, n, total int, kind, msg string) {
	fmt.Fprintf(w, "[%d/%d] %-12s %s\n", n, total, kind, msg)
}

func stageDetail(w io.Writer, n, total int, format string, args ...any) {
	prefix := fmt.Sprintf("[%d/%d] ", n, total)
	indent := strings.Repeat(" ", len(prefix))
	fmt.Fprintf(w, indent+format+"\n", args...)
}

func printInstallDryRun(stdout, stderr io.Writer, item installplan.Item, goos, goarch string, dest remoteInstallDest, remote installRemote, home string) error {
	envPrefix := fmt.Sprintf("GOOS=%s GOARCH=%s", goos, goarch)
	if goos != runtime.GOOS || goarch != runtime.GOARCH {
		envPrefix += " CGO_ENABLED=0"
	}
	switch item.Method {
	case installplan.MethodGoInstall:
		stagePrint(stderr, 2, 4, "build", goos+"/"+goarch)
		stageDetail(stderr, 2, 4, "would: %s go build -o <staging>/%s %s", envPrefix, item.BinName, item.RelPath)
	case installplan.MethodGoRunInstall:
		stagePrint(stderr, 2, 4, "build", goos+"/"+goarch)
		stageDetail(stderr, 2, 4, "would: INSTALL_GOOS=%s INSTALL_GOARCH=%s INSTALL_TO_DIR=<staging> go run %s", goos, goarch, item.RelPath)
	default:
		return fmt.Errorf("unknown install method %q", item.Method)
	}

	stagePrint(stderr, 3, 4, "upload", "→ "+dest.Primary)
	if dest.FromPATH {
		stageDetail(stderr, 3, 4, "would: upload %s → %s (existing on PATH)", item.BinName, dest.Primary)
	} else {
		stageDetail(stderr, 3, 4, "would: upload %s → %s (default ~/.local/bin)", item.BinName, dest.Primary)
	}
	for _, extra := range dest.Extras {
		stageDetail(stderr, 3, 4, "would: also %s", extra)
	}

	if destWritesLocalBin(dest, home) {
		stagePrint(stderr, 4, 4, "path", "ensure ~/.local/bin on PATH")
		if err := ensureRemoteLocalBinPATH(remote, home, true, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "warning: probe PATH rc: %v\n", err)
		}
	} else {
		stagePrint(stderr, 4, 4, "path", "skipped (not writing ~/.local/bin)")
	}

	fmt.Fprintln(stderr)
	fmt.Fprintf(stdout, "would: install %s → %s (%s/%s)\n", item.BinName, dest.Primary, goos, goarch)
	return nil
}
