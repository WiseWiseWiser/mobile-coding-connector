package agentcli

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
	"golang.org/x/term"
)

// defaultEditWorkDir is the staging root for `edit`: the remote file is
// downloaded to <root>/<remote-path>, edited there, and written back.
const defaultEditWorkDir = "/tmp/remote-agent-edit"

func editHelpFor(p Profile) string {
	name := p.Name
	if name == "" {
		name = "remote-agent"
	}
	return fmt.Sprintf(`Usage: %s edit <REMOTE_PATH> [options]

Edit a remote file with a local editor, then write it back with conflict
detection.

The file is staged at %s/<REMOTE_PATH> on this machine. If the remote file
does not exist the staged copy starts empty. When the editor exits, the staged
content is uploaded only when the remote file still has the md5 recorded at
download time; otherwise the save fails, nothing is written, and the staged
copy is kept so the conflict can be resolved by hand.

A staged copy that already matches the remote content is reused, so editing the
same file again transfers nothing.

Arguments:
  REMOTE_PATH          Path on the server. May use ~/ for the server home;
                       relative paths resolve against the server home.

Options:
  --editor CMD         Editor to run. Default: $VISUAL, then $EDITOR, then vim.
                       CMD may include arguments, e.g. "code --wait". Known GUI
                       editors get their wait flag automatically (code →
                       --wait, subl/mate/atom → -w).
  --remember-flags     Save this run's flags (--editor, --work-dir) as the
                       default for later '%s edit' runs. With no other flags
                       given, the remembered flags are cleared.
  --work-dir DIR       Staging root (default: %s).
  -h, --help           Show this help.

Examples:
  %s edit /etc/nginx/nginx.conf
  %s edit '~/notes/todo.md' --editor=nano
  %s edit /etc/app/config.yaml --editor=code --remember-flags

Notes:
  When the staged copy already matches the remote content (same md5), the
  download is skipped and the staged copy is used as-is. If the server cannot
  report the remote digest, the full copy is downloaded as before.
`, name, defaultEditWorkDir, name, defaultEditWorkDir, name, name, name)
}

type editOptions struct {
	editor        string
	workDir       string
	rememberFlags bool
}

// runEdit implements `edit <REMOTE_PATH> [options]`.
func runEdit(cli *client.Client, args []string) error {
	if cli == nil {
		return fmt.Errorf("edit requires a server; see '%s edit --help'", active.Name)
	}

	opts, remembered, rest, err := parseEditOptions(args)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("edit requires <REMOTE_PATH>; see '%s edit --help'", active.Name)
	}
	if len(rest) > 1 {
		return fmt.Errorf("edit takes exactly one <REMOTE_PATH>, got %d: %v", len(rest), rest)
	}

	if len(remembered) > 0 {
		fmt.Printf("Using remembered flags: %s\n", strings.Join(remembered, " "))
	}
	if opts.rememberFlags {
		saved := extractEditFlags(args)
		if err := saveRememberedEditFlags(saved); err != nil {
			return err
		}
		if len(saved) == 0 {
			fmt.Println("Cleared remembered flags for edit")
		} else {
			fmt.Printf("Remembered flags for edit: %s\n", strings.Join(saved, " "))
		}
	}

	file := rest[0]
	remotePath, err := cli.ResolveRemoteFilePath(file)
	if err != nil {
		return err
	}

	// Ask for the remote digest together with existence: a staged copy that
	// already hashes to it can be reused, so a repeat edit transfers nothing.
	info, err := cli.CheckPathMD5(remotePath)
	probeFailed := err != nil
	if probeFailed {
		// The digest is an optimization only. Never fail an edit because the
		// server could not produce it (proxy timeout, hashing error, old build).
		fmt.Fprintf(os.Stderr, "warning: remote digest unavailable (%v); downloading the full copy\n", err)
		info, err = cli.CheckPath(remotePath)
		if err != nil {
			return fmt.Errorf("failed to check remote path %s: %w", remotePath, err)
		}
	}
	if info.IsDir {
		return fmt.Errorf("remote path %s is a directory, not a file", remotePath)
	}
	// One warning per run: after a failed probe the fallback check cannot report
	// a digest either, and repeating the reason would only add noise.
	if info.Exists && info.MD5 == "" && !probeFailed {
		fmt.Fprintln(os.Stderr, "warning: server did not report the remote digest; downloading the full copy")
	}

	// Resolve and validate the editor before downloading: a missing binary or a
	// terminal editor without a tty must fail before staging anything.
	editor, err := resolveEditor(opts.editor)
	if err != nil {
		return err
	}
	if err := checkEditorUsable(editor); err != nil {
		return err
	}

	workDir := opts.workDir
	if workDir == "" {
		workDir = defaultEditWorkDir
	}
	stagedPath := filepath.Join(workDir, remotePath)
	if err := os.MkdirAll(filepath.Dir(stagedPath), 0700); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	baseMD5 := ""
	if info.Exists && stagedCopyMatches(stagedPath, info.MD5) {
		fmt.Printf("Skipped download: %s already matches the remote (md5 %s)\n", stagedPath, info.MD5)
		baseMD5 = info.MD5
	} else {
		if info.Exists {
			fmt.Printf("Downloading %s -> %s (%s)\n", file, stagedPath, formatSize(info.Size))
			if _, err := cli.DownloadFile(remotePath, stagedPath, client.DownloadOptions{NoResume: true}, nil); err != nil {
				return fmt.Errorf("failed to download %s: %w", remotePath, err)
			}
		} else {
			fmt.Printf("Remote %s is missing; starting from an empty file\n", file)
			if err := os.WriteFile(stagedPath, nil, 0600); err != nil {
				return fmt.Errorf("failed to create staged file: %w", err)
			}
		}
		// Staged copies can hold secrets (dotfiles, .env, private keys): keep
		// them readable only by this user regardless of the download default.
		_ = os.Chmod(stagedPath, 0600)

		baseMD5, err = fileMD5Hex(stagedPath)
		if err != nil {
			return fmt.Errorf("failed to hash staged copy: %w", err)
		}
	}

	fmt.Printf("Opening %s %s\n", editor.String(), stagedPath)
	if err := runEditor(editor, stagedPath); err != nil {
		return err
	}

	newMD5, err := fileMD5Hex(stagedPath)
	if err != nil {
		return fmt.Errorf("staged copy %s is gone; nothing was uploaded", stagedPath)
	}
	if newMD5 == baseMD5 {
		fmt.Println("file not changed")
		return nil
	}

	result, err := cli.WriteFileConditional(remotePath, stagedPath, baseMD5)
	if err != nil {
		var conflict *client.FileConflictError
		if errors.As(err, &conflict) {
			if conflict.CurrentMD5 == newMD5 {
				// Someone else produced exactly our bytes: the desired state is
				// already on the server, so there is nothing to resolve.
				fmt.Printf("Remote %s already matches your edits; nothing written\n", remotePath)
				return nil
			}
			return editConflictError(remotePath, stagedPath, conflict)
		}
		var unsupported *client.UnsupportedWriteError
		if errors.As(err, &unsupported) {
			return fmt.Errorf("%w\n  hint: upgrade the server ('%s server upgrade'), or push the staged copy with '%s upload %s %s'",
				err, active.Name, active.Name, stagedPath, file)
		}
		return err
	}

	if result.ResolvedPath != "" && result.ResolvedPath != result.Path {
		fmt.Fprintf(os.Stderr, "warning: %s is a symlink; wrote %s\n", result.Path, result.ResolvedPath)
	}
	verb := "Saved"
	if result.Created {
		verb = "Created"
	}
	fmt.Printf("%s %s (%s, md5 %s)\n", verb, result.Path, formatSize(result.Size), result.MD5)
	return nil
}

// parseEditOptions parses edit flags. Remembered flags are applied first so an
// explicit command-line flag always wins (the parser keeps the last value).
func parseEditOptions(args []string) (*editOptions, []string, []string, error) {
	opts := &editOptions{}
	newParser := func() *flags.Builder {
		return flags.
			String("--editor", &opts.editor).
			String("--work-dir", &opts.workDir).
			Bool("--remember-flags", &opts.rememberFlags).
			HelpFunc("-h,--help", func() { fmt.Print(editHelpFor(active)) }).
			HelpNoExit()
	}

	remembered, err := loadRememberedEditFlags()
	if err != nil {
		return nil, nil, nil, err
	}
	applied := remembered
	if len(remembered) > 0 {
		if _, err := newParser().Parse(remembered); err != nil && !errors.Is(err, flags.ErrHelp) {
			// A remembered flag may no longer exist after a CLI upgrade; drop
			// the memory instead of failing every later run.
			fmt.Fprintf(os.Stderr, "warning: ignoring remembered edit flags %v: %v\n", remembered, err)
			opts.editor, opts.workDir = "", ""
			applied = nil
		}
	}

	rest, err := newParser().Parse(args)
	if err != nil {
		return nil, nil, nil, err
	}
	return opts, applied, rest, nil
}

// extractEditFlags returns the edit flags of an invocation in a form that can
// be replayed later, dropping --remember-flags itself.
func extractEditFlags(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--editor" || arg == "--work-dir":
			if i+1 < len(args) {
				out = append(out, arg, args[i+1])
				i++
			}
		case strings.HasPrefix(arg, "--editor=") || strings.HasPrefix(arg, "--work-dir="):
			out = append(out, arg)
		}
	}
	return out
}

func loadRememberedEditFlags() ([]string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return cfg.RememberedFlags[editRememberKey], nil
}

func saveRememberedEditFlags(remembered []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &agentConfig{}
	}
	if len(remembered) == 0 {
		delete(cfg.RememberedFlags, editRememberKey)
	} else {
		if cfg.RememberedFlags == nil {
			cfg.RememberedFlags = map[string][]string{}
		}
		cfg.RememberedFlags[editRememberKey] = remembered
	}
	return saveConfig(cfg)
}

// editRememberKey is the remembered_flags map key for this command.
const editRememberKey = "edit"

// editorCommand is a resolved editor invocation.
type editorCommand struct {
	name string
	args []string
}

func (e editorCommand) String() string {
	return strings.Join(append([]string{e.name}, e.args...), " ")
}

// editorWaitArgs maps GUI editors that detach by default to the flag that
// makes them block until the file is closed.
var editorWaitArgs = map[string][]string{
	"code":          {"--wait"},
	"code-insiders": {"--wait"},
	"codium":        {"--wait"},
	"subl":          {"-w"},
	"mate":          {"-w"},
	"atom":          {"-w"},
}

// terminalEditors are editors that need an interactive terminal on stdin.
var terminalEditors = map[string]bool{
	"vim":    true,
	"vi":     true,
	"nvim":   true,
	"nano":   true,
	"pico":   true,
	"micro":  true,
	"hx":     true,
	"helix":  true,
	"joe":    true,
	"mcedit": true,
}

// resolveEditor turns the --editor flag (or $VISUAL/$EDITOR, or vim) into a
// command. The value may contain arguments separated by spaces.
func resolveEditor(flagValue string) (editorCommand, error) {
	spec := strings.TrimSpace(flagValue)
	if spec == "" {
		spec = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if spec == "" {
		spec = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if spec == "" {
		spec = "vim"
	}

	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return editorCommand{}, fmt.Errorf("empty editor command")
	}
	editor := editorCommand{name: fields[0], args: fields[1:]}
	if wait, ok := editorWaitArgs[filepath.Base(editor.name)]; ok && !hasAnyArg(editor.args, wait) {
		editor.args = append(editor.args, wait...)
	}
	return editor, nil
}

func hasAnyArg(args, want []string) bool {
	for _, a := range args {
		for _, w := range want {
			if a == w {
				return true
			}
		}
	}
	return false
}

// checkEditorUsable fails fast when the editor is missing or cannot work in
// this environment, instead of leaving the user (or an agent) in a hung or
// confusing editor session.
func checkEditorUsable(editor editorCommand) error {
	if _, err := exec.LookPath(editor.name); err != nil {
		return fmt.Errorf("editor %q not found in PATH; install it or pass --editor=<command>", editor.name)
	}
	if terminalEditors[filepath.Base(editor.name)] && !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("editor %q needs a terminal on stdin; pass a GUI editor (e.g. --editor=code) or run %s edit from an interactive shell",
			editor.name, active.Name)
	}
	return nil
}

// runEditor runs the editor on the staged file with this process's stdio, so
// terminal editors get the real tty.
func runEditor(editor editorCommand, path string) error {
	argv := append(append([]string{}, editor.args...), path)
	cmd := exec.Command(editor.name, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("editor exited with status %d; nothing was uploaded (staged copy: %s)",
				exitErr.ExitCode(), path)
		}
		return fmt.Errorf("failed to run editor %q: %w", editor.name, err)
	}
	return nil
}

// editConflictError renders the resolution recipe for a rejected save.
func editConflictError(remotePath, stagedPath string, conflict *client.FileConflictError) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s changed on the server while you were editing; nothing was written\n", remotePath)
	fmt.Fprintf(&b, "  downloaded md5: %s\n", conflict.ExpectedMD5)
	fmt.Fprintf(&b, "  current  md5: %s\n", conflict.CurrentMD5)
	if !conflict.CurrentExists {
		fmt.Fprintf(&b, "  the remote file was deleted\n")
	}
	fmt.Fprintf(&b, "  your edits are kept at: %s\n", stagedPath)
	fmt.Fprintf(&b, "hint: reconcile, then push your copy:\n")
	fmt.Fprintf(&b, "        %s download %s %s.remote\n", active.Name, remotePath, stagedPath)
	fmt.Fprintf(&b, "        %s upload %s %s\n", active.Name, stagedPath, remotePath)
	return errors.New(b.String())
}

// stagedCopyMatches reports whether the staged copy can be used as the edit base
// without downloading: the remote digest must be known (so an absent path or an
// old server never reuses anything) and the staged bytes must hash to it. Digest
// equality means the staged copy is byte-identical to the remote content, so the
// save precondition stays exactly the same.
func stagedCopyMatches(stagedPath, remoteMD5 string) bool {
	if remoteMD5 == "" {
		return false
	}
	stagedMD5, err := fileMD5Hex(stagedPath)
	if err != nil {
		return false
	}
	return stagedMD5 == remoteMD5
}

// fileMD5Hex returns the hex md5 of a file's contents.
func fileMD5Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
