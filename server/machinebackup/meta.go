package machinebackup

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/tools"
)

const (
	metaConfigName    = "config.json"
	metaInstalledName = "installed.json"
	metaEnvName       = "ENV"
	machineBakSuffix  = ".machine.bak"
)

var backupMetaFiles = []string{metaConfigName, metaInstalledName, metaEnvName}

// InstalledToolsSnapshot is written to .backup/installed.json.
type InstalledToolsSnapshot struct {
	CapturedAt time.Time          `json:"captured_at"`
	Tools      []InstalledToolRef `json:"tools"`
}

// InstalledToolRef is one installed binary from the tools registry.
type InstalledToolRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path"`
}

var buildInstalledToolsSnapshotFn = buildInstalledToolsSnapshotFromRegistry

func buildInstalledToolsSnapshotFromRegistry() ([]byte, error) {
	installed := tools.SnapshotInstalledTools()
	snap := InstalledToolsSnapshot{CapturedAt: time.Now().UTC()}
	for _, tool := range installed {
		snap.Tools = append(snap.Tools, InstalledToolRef{
			Name:    tool.Name,
			Version: tool.Version,
			Path:    tool.Path,
		})
	}
	sort.Slice(snap.Tools, func(i, j int) bool { return snap.Tools[i].Name < snap.Tools[j].Name })
	return json.MarshalIndent(snap, "", "  ")
}

func buildEnvSnapshot() []byte {
	env := os.Environ()
	sort.Strings(env)
	return []byte(strings.Join(env, "\n") + "\n")
}

func writeBackupMeta(tw *tar.Writer, home string, rules ExclusionRules, gitRepos *GitRepoWorktreesSnapshot, gitSkipped bool) error {
	payloads := map[string][]byte{}
	cfg, err := json.MarshalIndent(rules.EffectiveExclusionConfig(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup config: %w", err)
	}
	payloads[metaConfigName] = cfg

	installed, err := buildInstalledToolsSnapshotFn()
	if err != nil {
		return fmt.Errorf("build installed.json: %w", err)
	}
	payloads[metaInstalledName] = installed
	payloads[metaEnvName] = buildEnvSnapshot()
	if !gitSkipped && gitRepos != nil {
		gitData, err := marshalGitReposSnapshot(gitRepos)
		if err != nil {
			return fmt.Errorf("marshal git-repo-worktrees.json: %w", err)
		}
		payloads[metaGitReposName] = gitData
	}

	tailscaleSnap, tailscaleIncluded, err := captureTailscaleConfigForHome(home)
	if err != nil {
		return fmt.Errorf("build tailscale-config.json: %w", err)
	}
	if tailscaleIncluded && tailscaleSnap != nil {
		tailscaleData, err := marshalTailscaleConfigSnapshot(tailscaleSnap)
		if err != nil {
			return fmt.Errorf("marshal tailscale-config.json: %w", err)
		}
		payloads[metaTailscaleName] = tailscaleData
	}

	metaFiles := append([]string(nil), backupMetaFiles...)
	if !gitSkipped && gitRepos != nil {
		metaFiles = append(metaFiles, metaGitReposName)
	}
	if tailscaleIncluded && tailscaleSnap != nil {
		metaFiles = append(metaFiles, metaTailscaleName)
	}

	cloudflaredSnap, cloudflaredIncluded, err := captureCloudflaredConfigForHome(home)
	if err != nil {
		return fmt.Errorf("build cloudflared-config.json: %w", err)
	}
	if cloudflaredIncluded && cloudflaredSnap != nil {
		cloudflaredData, err := marshalCloudflaredConfigSnapshot(cloudflaredSnap)
		if err != nil {
			return fmt.Errorf("marshal cloudflared-config.json: %w", err)
		}
		payloads[metaCloudflaredName] = cloudflaredData
	}
	if cloudflaredIncluded && cloudflaredSnap != nil {
		metaFiles = append(metaFiles, metaCloudflaredName)
	}

	systemdSnap, systemdIncluded, err := captureSystemdServicesForHome(home)
	if err != nil {
		return fmt.Errorf("build systemd-services.json: %w", err)
	}
	if systemdIncluded && systemdSnap != nil {
		systemdData, err := marshalSystemdServicesSnapshot(systemdSnap)
		if err != nil {
			return fmt.Errorf("marshal systemd-services.json: %w", err)
		}
		payloads[metaSystemdServicesName] = systemdData
	}
	if systemdIncluded && systemdSnap != nil {
		metaFiles = append(metaFiles, metaSystemdServicesName)
	}

	metaHome := filepath.Join(home, backupMetaDir)
	for _, name := range metaFiles {
		existing := filepath.Join(metaHome, name)
		if data, err := os.ReadFile(existing); err == nil {
			bakPath := backupMetaDir + "/" + name + machineBakSuffix
			if err := writeTarBytes(tw, bakPath, 0644, data); err != nil {
				return err
			}
		}
		if err := writeTarBytes(tw, backupMetaDir+"/"+name, 0644, payloads[name]); err != nil {
			return err
		}
	}
	return nil
}

func isBackupMetaSnapshot(rel string) bool {
	rel = normalizeRelPath(rel)
	switch rel {
	case backupMetaDir + "/" + metaConfigName,
		backupMetaDir + "/" + metaInstalledName,
		backupMetaDir + "/" + metaEnvName,
		backupMetaDir + "/" + metaGitReposName,
		backupMetaDir + "/" + metaTailscaleName,
		backupMetaDir + "/" + metaCloudflaredName,
		backupMetaDir + "/" + metaSystemdServicesName:
		return true
	default:
		return false
	}
}

func isBackupMachineBak(rel string) (orig string, ok bool) {
	rel = normalizeRelPath(rel)
	prefix := backupMetaDir + "/"
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	base := strings.TrimPrefix(rel, prefix)
	if !strings.HasSuffix(base, machineBakSuffix) {
		return "", false
	}
	origName := strings.TrimSuffix(base, machineBakSuffix)
	if origName == "" {
		return "", false
	}
	return backupMetaDir + "/" + origName, true
}

func resolveRestoreTarget(rel string) (target string, skip bool) {
	rel = normalizeRelPath(rel)
	if rel == "" {
		return "", true
	}
	if isBackupMetaSnapshot(rel) {
		return "", true
	}
	if orig, ok := isBackupMachineBak(rel); ok {
		return orig, false
	}
	if rel == backupMetaDir || strings.HasPrefix(rel, backupMetaDir+"/") {
		return "", true
	}
	return rel, false
}