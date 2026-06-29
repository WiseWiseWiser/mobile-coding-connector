package agentcli

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
)

const projectPullLocalHelp = `Usage: remote-agent project pull-local <project-id-or-name-or-dir> [options]

Copy dirty remote project changes into a new local git worktree.

Options:
  --local-path PATH       Local git repository (overrides saved binding)
  --no-truncate-remote    Keep remote dirty state after a successful pull
  --dry-run               Print the pull plan without making changes
  --include-file PATH     Include PATH in pull even if over 1 MB (repeatable)
  --max-size SIZE         Max package size e.g. 64M, 100M (default 64M)
  -h, --help              Show this help message
`

func runProjectPullLocal(resolve func() (*client.Client, error), args []string) error {
	var localPathFlag string
	var noTruncate bool
	var dryRun bool
	var includeFiles []string
	var maxSizeFlag string

	args, err := flags.
		String("--local-path", &localPathFlag).
		Bool("--no-truncate-remote", &noTruncate).
		Bool("--dry-run", &dryRun).
		StringSlice("--include-file", &includeFiles).
		String("--max-size", &maxSizeFlag).
		Help("-h,--help", projectPullLocalHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("project pull-local requires exactly 1 argument <project-id-or-name-or-dir>")
	}

	maxSizeBytes, err := parsePullLocalMaxSize(maxSizeFlag)
	if err != nil {
		return err
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	project, err := resolveProjectTarget(cli, args[0])
	if err != nil {
		return err
	}

	if project.GitStatus.IsClean {
		return fmt.Errorf("nothing to pull: remote worktree is clean")
	}

	localPath, savedBinding, err := resolvePullLocalPath(cli, project, strings.TrimSpace(localPathFlag))
	if err != nil {
		return err
	}

	localOrigin, err := localGitOriginURL(localPath)
	if err != nil {
		return err
	}
	remoteOrigin, err := remoteGitOriginURL(cli, project.Dir)
	if err != nil {
		return err
	}
	if !sameGitOrigin(localOrigin, remoteOrigin) {
		return fmt.Errorf("git origin mismatch: local %q vs remote %q", localOrigin, remoteOrigin)
	}

	branch, err := resolveRemoteBranch(cli, project)
	if err != nil {
		return err
	}
	worktreePath, err := allocateWorktreeDir(cli.Server, project.Name, project.Dir, branch)
	if err != nil {
		return err
	}

	pullReq := client.PullLocalRequest{
		Dir:          project.Dir,
		IncludeFiles: includeFiles,
		MaxSizeBytes: maxSizeBytes,
	}

	plan, err := cli.PullLocal(pullReq)
	if err != nil {
		return err
	}

	if dryRun {
		printPullLocalDryRunPlan(project, localPath, worktreePath, plan, noTruncate)
		return nil
	}

	pkgBody, err := cli.PullLocalPackage(pullReq)
	if err != nil {
		return err
	}
	defer pkgBody.Close()

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return err
	}
	if err := cmd.Dir(localPath).Run("git", "worktree", "add", "--detach", worktreePath, plan.Commit); err != nil {
		return fmt.Errorf("create local worktree: %w", err)
	}

	if err := applyPullLocalPackage(pkgBody, worktreePath); err != nil {
		return err
	}

	if !noTruncate {
		if err := cli.PullLocalTruncate(project.Dir, plan.Commit); err != nil {
			return err
		}
	}

	fmt.Printf("Pulled dirty state from %s into worktree %s\n", project.Dir, worktreePath)
	if !savedBinding && localPathFlag == "" {
		fmt.Printf("Saved local binding %s\n", localPath)
	}
	if noTruncate {
		fmt.Println("Remote repository was not truncated (--no-truncate-remote).")
	} else {
		fmt.Println("Remote repository was reset to a clean state.")
	}
	return nil
}

func printPullLocalDryRunPlan(project *client.ProjectInfo, localPath, worktreePath string, plan *client.PullLocalPlan, noTruncate bool) {
	fmt.Printf("dry-run: pull-local plan for %s (%s)\n\n", project.Name, project.ID)
	fmt.Printf("  remote dir:   %s\n", project.Dir)
	fmt.Printf("  commit:       %s\n", plan.Commit)
	fmt.Printf("  branch:       %s\n", plan.Branch)
	fmt.Printf("  local repo:   %s\n", localPath)
	fmt.Printf("  worktree:     %s\n", worktreePath)
	fmt.Printf("  diff bytes:   %d\n", plan.EstimatedBytes)
	fmt.Printf("  untracked:    %d\n", plan.UntrackedFiles)
	if noTruncate {
		fmt.Printf("  remote:       would keep dirty state (--no-truncate-remote)\n")
	} else {
		fmt.Printf("  remote:       would reset --hard and clean -fd\n")
	}
}

func applyPullLocalPackage(r io.Reader, worktreePath string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("read pull package: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	var patchData []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		switch {
		case hdr.Name == "patch.diff":
			patchData, err = io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("read patch.diff: %w", err)
			}
		case strings.HasPrefix(hdr.Name, "untracked/"):
			rel := strings.TrimPrefix(hdr.Name, "untracked/")
			rel = filepath.FromSlash(rel)
			dest := filepath.Join(worktreePath, rel)
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return fmt.Errorf("skip tar entry %s: %w", hdr.Name, err)
			}
		}
	}

	if len(patchData) > 0 && strings.TrimSpace(string(patchData)) != "" {
		if err := applyGitDiffInDir(worktreePath, string(patchData)); err != nil {
			return err
		}
	}
	return nil
}

func parsePullLocalMaxSize(flag string) (int64, error) {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return 0, nil
	}
	s := strings.TrimSpace(flag)
	mult := int64(1)
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "gb"):
		mult = 1 << 30
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "g"):
		mult = 1 << 30
		s = s[:len(s)-1]
	case strings.HasSuffix(lower, "mb"):
		mult = 1 << 20
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "m"):
		mult = 1 << 20
		s = s[:len(s)-1]
	case strings.HasSuffix(lower, "kb"):
		mult = 1 << 10
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "k"):
		mult = 1 << 10
		s = s[:len(s)-1]
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid --max-size %q", flag)
	}
	return n * mult, nil
}

func resolveRemoteBranch(cli *client.Client, project *client.ProjectInfo) (string, error) {
	branch := strings.TrimSpace(project.GitStatus.Branch)
	if branch == "" || branch == "(detached)" {
		for _, candidate := range []string{"main", "master"} {
			if remoteRefExists(cli, project.Dir, "refs/remotes/origin/"+candidate) {
				return candidate, nil
			}
		}
		return "detached", nil
	}
	if branch == "master" && remoteRefExists(cli, project.Dir, "refs/remotes/origin/main") {
		return "main", nil
	}
	return branch, nil
}

func remoteRefExists(cli *client.Client, dir, ref string) bool {
	_, code, err := remoteGitOutput(cli, dir, "show-ref", "--verify", ref)
	return err == nil && code == 0
}

func resolvePullLocalPath(cli *client.Client, project *client.ProjectInfo, flagPath string) (string, bool, error) {
	if flagPath != "" {
		abs, err := filepath.Abs(flagPath)
		if err != nil {
			return "", false, err
		}
		return abs, false, nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return "", false, err
	}
	remoteDir := filepath.Clean(project.Dir)
	if path, ok := findProjectBinding(cfg, cli.Server, remoteDir); ok {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", true, err
		}
		return abs, true, nil
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("No local binding for %s.\n", project.Name)
		fmt.Printf("Enter local git repository path: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", false, fmt.Errorf("read local path: %w", err)
		}
		abs, err := filepath.Abs(strings.TrimSpace(line))
		if err != nil {
			return "", false, err
		}
		if cfg == nil {
			cfg = &agentConfig{}
		}
		if err := upsertProjectBinding(cfg, cli.Server, remoteDir, abs); err != nil {
			return "", false, err
		}
		if err := saveConfig(cfg); err != nil {
			return "", false, err
		}
		return abs, false, nil
	}

	return "", false, fmt.Errorf("no project binding for %s; run project bind-local or pass --local-path (non-interactive)", project.Name)
}

func applyGitDiffInDir(dir, diff string) error {
	tmp, err := os.CreateTemp("", "pull-local-diff-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if diff != "" && !strings.HasSuffix(diff, "\n") {
		diff += "\n"
	}
	if _, err := tmp.WriteString(diff); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return cmd.Dir(dir).Run("git", "apply", "--whitespace=nowarn", tmpPath)
}