package synccmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PreferredUnisonVersion is the pinned Unison version install aims for.
const PreferredUnisonVersion = "2.54.0"

// DefaultRemoteUnisonPath is used when RemoteTargetPath is empty.
// Production remote install still requires a RemoteEnsure hook (or later
// --serve/upload wiring); this is only the default target path argument.
const DefaultRemoteUnisonPath = ".local/bin/unison"

// InstallOpts configures Install with injectable ensure hooks (parallel-safe).
type InstallOpts struct {
	// Scope is "local", "remote", or "both". Empty → "both".
	Scope string

	// LocalEnsure ensures local unison is installed/available; returns version.
	// Nil → product probes PATH via LookPath + `unison -version`.
	LocalEnsure func() (version string, err error)

	// RemoteEnsure places/ensures remote binary at targetPath; returns version.
	// Nil → clear error (remote install needs --serve/upload wiring).
	RemoteEnsure func(targetPath string) (version string, err error)

	// WhichLocal optionally probes existing local binary (path + version).
	// Nil OK; not required by thin P4 leaves.
	WhichLocal func() (path, version string, err error)

	// RemoteTargetPath is passed to RemoteEnsure. Empty → product default path.
	RemoteTargetPath string

	Stdout io.Writer
	Stderr io.Writer
}

// InstallReport is the structured install outcome.
type InstallReport struct {
	LocalVersion  string
	RemoteVersion string
	LocalOK       bool
	RemoteOK      bool
	Messages      []string // optional human lines; may mention PreferredUnisonVersion
}

// Install ensures Unison is present for local and/or remote sides per Scope.
func Install(opts InstallOpts) (InstallReport, error) {
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope == "" {
		scope = "both"
	}

	var rep InstallReport
	rep.Messages = append(rep.Messages, fmt.Sprintf("preferred unison version: %s", PreferredUnisonVersion))

	wantLocal := scope == "local" || scope == "both"
	wantRemote := scope == "remote" || scope == "both"
	if !wantLocal && !wantRemote {
		return rep, fmt.Errorf("install: unknown scope %q (want local|remote|both)", opts.Scope)
	}

	var firstErr error

	if wantLocal {
		ensure := opts.LocalEnsure
		if ensure == nil {
			ensure = defaultLocalEnsure
		}
		ver, err := ensure()
		if err != nil {
			rep.LocalOK = false
			rep.Messages = append(rep.Messages, fmt.Sprintf("local install failed: %v", err))
			if firstErr == nil {
				firstErr = fmt.Errorf("install local: %w", err)
			}
		} else {
			rep.LocalOK = true
			rep.LocalVersion = strings.TrimSpace(ver)
			rep.Messages = append(rep.Messages, fmt.Sprintf("local unison ok: %s", rep.LocalVersion))
		}
	}

	if wantRemote {
		target := opts.RemoteTargetPath
		if target == "" {
			target = defaultRemoteTargetPath()
		}
		ensure := opts.RemoteEnsure
		if ensure == nil {
			ensure = defaultRemoteEnsure
		}
		ver, err := ensure(target)
		if err != nil {
			rep.RemoteOK = false
			rep.Messages = append(rep.Messages, fmt.Sprintf("remote install failed: %v", err))
			if firstErr == nil {
				firstErr = fmt.Errorf("install remote: %w", err)
			}
		} else {
			rep.RemoteOK = true
			rep.RemoteVersion = strings.TrimSpace(ver)
			rep.Messages = append(rep.Messages, fmt.Sprintf("remote unison ok: %s at %s", rep.RemoteVersion, target))
		}
	}

	return rep, firstErr
}

func defaultRemoteTargetPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return DefaultRemoteUnisonPath
	}
	return filepath.Join(home, DefaultRemoteUnisonPath)
}

// defaultLocalEnsure probes PATH for unison and parses `unison -version`.
func defaultLocalEnsure() (string, error) {
	path, err := exec.LookPath("unison")
	if err != nil {
		return "", fmt.Errorf("local unison not found on PATH (preferred %s): %w", PreferredUnisonVersion, err)
	}
	cmd := exec.Command(path, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("local unison -version failed: %w", err)
	}
	ver := parseUnisonVersion(string(out))
	if ver == "" {
		return "", fmt.Errorf("could not parse local unison version from %q", strings.TrimSpace(string(out)))
	}
	return ver, nil
}

// defaultRemoteEnsure is the production stub when RemoteEnsure is not injected.
func defaultRemoteEnsure(targetPath string) (string, error) {
	_ = targetPath
	return "", fmt.Errorf("remote install requires --serve and upload")
}

// unisonVersionLabeledRe prefers "unison version 2.54.0" so SSH banners
// like "Permanently added '[127.0.0.1]:port'" are not mistaken for Unison.
var unisonVersionLabeledRe = regexp.MustCompile(`(?i)unison\s+version\s+(\d+\.\d+(?:\.\d+)?)`)

// unisonVersionBareRe matches a lone semver-ish token (fallback only).
var unisonVersionBareRe = regexp.MustCompile(`\b(\d+\.\d+\.\d+)\b`)

func parseUnisonVersion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 1) Prefer explicit "unison version X.Y.Z" anywhere.
	if m := unisonVersionLabeledRe.FindStringSubmatch(s); len(m) >= 2 {
		return m[1]
	}
	// 2) Fallback: first X.Y.Z on a line that mentions unison (not 127.0.0.1).
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(strings.ToLower(line), "unison") {
			continue
		}
		if m := unisonVersionBareRe.FindStringSubmatch(line); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}
