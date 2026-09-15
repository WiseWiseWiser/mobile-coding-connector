package agentcli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	shelllocalbin "github.com/xhd2015/dot-pkgs/go-pkgs/shell/localbin"
)

type installRemote interface {
	Home() (string, error)
	LookCommand(name string) (string, error)
	GoEnv(name string) (string, error)
	CheckExists(remotePath string) (bool, error)
	UploadExec(localPath, remotePath string) error
	ReadFile(remotePath string) (data []byte, exists bool, err error)
	WriteFile(remotePath string, data []byte) error
}

type liveInstallRemote struct {
	cli *client.Client
}

func (r liveInstallRemote) Home() (string, error) {
	info, err := r.cli.GetHome()
	if err != nil {
		return "", err
	}
	home := strings.TrimRight(strings.TrimSpace(info.Home), "/")
	if home == "" {
		return "", fmt.Errorf("remote HOME is empty")
	}
	return home, nil
}

func (r liveInstallRemote) LookCommand(name string) (string, error) {
	for _, shell := range []string{"bash", "zsh"} {
		out, code, err := r.execCapture(shell, "-lc", "command -v "+name)
		if err != nil {
			continue
		}
		if code != 0 {
			continue
		}
		p := strings.TrimSpace(out)
		if p != "" {
			return p, nil
		}
	}
	return "", nil
}

func (r liveInstallRemote) GoEnv(name string) (string, error) {
	out, code, err := r.execCapture("bash", "-lc", "go env "+name)
	if err != nil || code != 0 {
		out, code, err = r.execCapture("zsh", "-lc", "go env "+name)
		if err != nil || code != 0 {
			return "", nil
		}
	}
	return strings.TrimSpace(out), nil
}

func (r liveInstallRemote) CheckExists(remotePath string) (bool, error) {
	info, err := r.cli.CheckPath(remotePath)
	if err != nil {
		return false, err
	}
	return info != nil && info.Exists && !info.IsDir, nil
}

func (r liveInstallRemote) UploadExec(localPath, remotePath string) error {
	_, err := r.cli.UploadFile(localPath, remotePath, client.UploadOptions{ChmodExec: true}, printUploadProgress)
	return err
}

func (r liveInstallRemote) ReadFile(remotePath string) ([]byte, bool, error) {
	exists, err := r.CheckExists(remotePath)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	tmp, err := os.CreateTemp("", "remote-agent-install-rc-*")
	if err != nil {
		return nil, false, err
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpName)
	if _, err := r.cli.DownloadFile(remotePath, tmpName, client.DownloadOptions{}, nil); err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (r liveInstallRemote) WriteFile(remotePath string, data []byte) error {
	tmp, err := os.CreateTemp("", "remote-agent-install-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	defer os.Remove(tmpName)
	_, err = r.cli.UploadFile(tmpName, remotePath, client.UploadOptions{}, nil)
	return err
}

func (r liveInstallRemote) execCapture(argv ...string) (stdout string, code int, err error) {
	var buf bytes.Buffer
	code, err = r.cli.Exec(client.ExecRequest{Argv: argv}, func(ev client.ExecEvent) {
		if ev.Type == "stdout" {
			buf.WriteString(ev.Data)
		}
	})
	return buf.String(), code, err
}

type remoteInstallDest struct {
	Primary       string
	Extras        []string
	FromPATH      bool
	SystemWarning string
}

func planRemoteInstallDest(binName, home, lookPath, gobin, gopath string) remoteInstallDest {
	localBin := path.Join(home, ".local", "bin", binName)
	var extras []string
	addExtra := func(p string) {
		if p == "" || p == lookPath {
			return
		}
		if !remotePathUnderHome(p, home) {
			return
		}
		for _, e := range extras {
			if e == p {
				return
			}
		}
		extras = append(extras, p)
	}

	var dest remoteInstallDest
	if lookPath != "" && remotePathUnderHome(lookPath, home) {
		dest.Primary = lookPath
		dest.FromPATH = true
	} else {
		if lookPath != "" {
			dest.SystemWarning = fmt.Sprintf("remote LookPath is %s; not overwriting a system path", lookPath)
		}
		dest.Primary = localBin
	}

	if dest.Primary != localBin {
		addExtra(localBin)
	}
	if gobin = strings.TrimSpace(gobin); gobin != "" {
		addExtra(path.Join(gobin, binName))
	}
	if gopath = strings.TrimSpace(gopath); gopath != "" {
		if i := strings.IndexByte(gopath, ':'); i >= 0 {
			gopath = gopath[:i]
		}
		addExtra(path.Join(gopath, "bin", binName))
	}

	// Extras are "existing copies"; the caller CheckExists-filters them.
	dest.Extras = extras
	return dest
}

func filterExistingExtras(dest remoteInstallDest, exists func(string) (bool, error)) (remoteInstallDest, error) {
	var kept []string
	for _, p := range dest.Extras {
		if p == dest.Primary {
			continue
		}
		ok, err := exists(p)
		if err != nil {
			return dest, err
		}
		if ok {
			kept = append(kept, p)
		}
	}
	dest.Extras = kept
	return dest, nil
}

func remotePathUnderHome(p, home string) bool {
	p = path.Clean(p)
	home = path.Clean(home)
	if p == home {
		return true
	}
	return strings.HasPrefix(p, home+"/")
}

func destWritesLocalBin(dest remoteInstallDest, home string) bool {
	localDir := path.Join(home, ".local", "bin")
	if path.Dir(dest.Primary) == localDir {
		return true
	}
	for _, e := range dest.Extras {
		if path.Dir(e) == localDir {
			return true
		}
	}
	return false
}

func ensureRemoteLocalBinPATH(r installRemote, home string, dryRun bool, stdout, stderr io.Writer) error {
	createFiles := []string{
		path.Join(home, ".bash_profile"),
		path.Join(home, ".bashrc"),
		path.Join(home, ".zshrc"),
	}
	existOnly := []string{
		path.Join(home, ".zprofile"),
		path.Join(home, ".profile"),
	}
	canonical := shelllocalbin.CheckerBlock()
	var updated []string
	for _, f := range createFiles {
		action, err := patchRemoteRC(r, f, canonical, true, dryRun)
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not update %s: %v\n", f, err)
			continue
		}
		printRemoteRC(stdout, stderr, f, action, dryRun)
		if action == "created" || action == "appended" || action == "replaced" {
			updated = append(updated, f)
		}
	}
	for _, f := range existOnly {
		exists, err := r.CheckExists(f)
		if err != nil || !exists {
			continue
		}
		action, err := patchRemoteRC(r, f, canonical, false, dryRun)
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not update %s: %v\n", f, err)
			continue
		}
		printRemoteRC(stdout, stderr, f, action, dryRun)
		if action == "created" || action == "appended" || action == "replaced" {
			updated = append(updated, f)
		}
	}
	if dryRun {
		return nil
	}
	if len(updated) > 0 {
		names := make([]string, len(updated))
		for i, f := range updated {
			names[i] = "~/" + path.Base(f)
		}
		fmt.Fprintf(stderr, "Added ~/.local/bin to PATH in %s\n", strings.Join(names, ", "))
		return nil
	}
	fmt.Fprintf(stderr, "PATH already includes ~/.local/bin\n")
	return nil
}

func printRemoteRC(stdout, stderr io.Writer, remotePath, action string, dryRun bool) {
	name := "~/" + path.Base(remotePath)
	if !dryRun {
		return
	}
	_ = stdout
	switch action {
	case "created":
		fmt.Fprintf(stderr, "would: create %s (PATH checker)\n", name)
	case "appended":
		fmt.Fprintf(stderr, "would: append %s (PATH checker)\n", name)
	case "replaced":
		fmt.Fprintf(stderr, "would: replace %s (PATH checker)\n", name)
	case "unchanged":
		fmt.Fprintf(stderr, "skip: %s (PATH checker already present)\n", name)
	}
}

func patchRemoteRC(r installRemote, remotePath, canonical string, create, dryRun bool) (action string, err error) {
	data, exists, err := r.ReadFile(remotePath)
	if err != nil {
		return "", err
	}
	if !exists {
		if !create {
			return "skipped_missing", nil
		}
		if dryRun {
			return "created", nil
		}
		if err := r.WriteFile(remotePath, []byte(canonical)); err != nil {
			return "", err
		}
		return "created", nil
	}
	newContent, action, _ := shelllocalbin.ApplyChecker(string(data), canonical)
	if action == "unchanged" || dryRun {
		return action, nil
	}
	if err := r.WriteFile(remotePath, []byte(newContent)); err != nil {
		return "", err
	}
	return action, nil
}
