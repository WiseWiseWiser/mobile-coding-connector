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

func writeBackupMeta(tw *tar.Writer, home string, rules ExclusionRules) error {
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

	metaHome := filepath.Join(home, backupMetaDir)
	for _, name := range backupMetaFiles {
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
		backupMetaDir + "/" + metaEnvName:
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