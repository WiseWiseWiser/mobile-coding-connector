package projectpull

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/xgo/support/cmd"
)

// BundleRequest is the JSON body for POST .../pull-local/bundle.
type BundleRequest struct {
	Dir string `json:"dir"`
}

// WriteBundle streams a git bundle that contains HEAD and enough history to
// materialize tip when origin lacks those commits.
func WriteBundle(w io.Writer, dir string) error {
	if err := validateDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "pull-local-bundle-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	createArgs := append([]string{"bundle", "create", tmpPath}, bundleCreateArgs(dir)...)
	if err := cmd.Dir(dir).Run("git", createArgs...); err != nil {
		// Fallback: bundle everything reachable from HEAD.
		if err2 := cmd.Dir(dir).Run("git", "bundle", "create", tmpPath, "HEAD"); err2 != nil {
			return fmt.Errorf("git bundle create: %v (fallback: %v)", err, err2)
		}
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func bundleCreateArgs(dir string) []string {
	// Prefer commits not on origin/<branch|main|master>; else full HEAD history.
	branchOut, err := cmd.Dir(dir).Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	branch := strings.TrimSpace(branchOut)
	if err != nil || branch == "" || branch == "HEAD" {
		return []string{"HEAD"}
	}
	for _, ref := range []string{"origin/" + branch, "origin/main", "origin/master"} {
		if _, err := cmd.Dir(dir).Output("git", "rev-parse", "--verify", ref); err == nil {
			// git bundle create <file> HEAD ^origin/main
			return []string{"HEAD", "^" + ref}
		}
	}
	return []string{"HEAD"}
}
