package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// chromeProfile identifies the actual profile containing an enabled Browser
// Agent extension. No cookies, history, or profile files are changed.
type chromeProfile struct{ Root, Name string }

type chromePreferenceFile struct {
	Extensions struct {
		Settings map[string]json.RawMessage `json:"settings"`
	} `json:"extensions"`
}

type chromeExtensionSetting struct {
	Path           string          `json:"path"`
	State          *int            `json:"state"`
	DisableReasons json.RawMessage `json:"disable_reasons"`
	Manifest       struct {
		Name string `json:"name"`
	} `json:"manifest"`
}

func chromeExtensionEnabled(ext chromeExtensionSetting) bool {
	if ext.State != nil && *ext.State != 1 {
		return false
	}
	reasons := strings.TrimSpace(string(ext.DisableReasons))
	return reasons == "" || reasons == "null" || reasons == "0" || reasons == "[]"
}

func profileHasAgent(dir string) (bool, error) {
	type record struct {
		path, name string
		disabled   bool
	}
	records := map[string]*record{}
	for _, name := range []string{"Preferences", "Secure Preferences"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("read Chrome profile preferences: %w", err)
		}
		var prefs chromePreferenceFile
		if err := json.Unmarshal(data, &prefs); err != nil {
			return false, fmt.Errorf("parse Chrome profile preferences: %w", err)
		}
		for id, raw := range prefs.Extensions.Settings {
			var ext chromeExtensionSetting
			if json.Unmarshal(raw, &ext) != nil {
				continue
			}
			r := records[id]
			if r == nil {
				r = &record{}
				records[id] = r
			}
			if !chromeExtensionEnabled(ext) {
				r.disabled = true
			}
			if ext.Path != "" {
				if r.path != "" && r.path != ext.Path {
					r.disabled = true
				}
				r.path = ext.Path
			}
			if ext.Manifest.Name != "" {
				if r.name != "" && r.name != ext.Manifest.Name {
					r.disabled = true
				}
				r.name = ext.Manifest.Name
			}
		}
	}
	count := 0
	for _, r := range records {
		if r.disabled || r.path == "" {
			continue
		}
		path := r.path
		if !filepath.IsAbs(path) {
			base := filepath.Join(dir, "Extensions")
			path = filepath.Join(base, path)
			rel, err := filepath.Rel(base, path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
		}
		data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
		if err != nil {
			continue
		}
		var manifest struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &manifest) == nil && manifest.Name == "Browser Agent" && (r.name == "" || r.name == manifest.Name) {
			count++
		}
	}
	if count > 1 {
		return false, fmt.Errorf("multiple enabled Browser Agent extensions in one Chrome profile; disable the older copy")
	}
	return count == 1, nil
}

func discoverChromeProfile(root, requested string) (chromeProfile, error) {
	if !filepath.IsAbs(root) {
		return chromeProfile{}, fmt.Errorf("Chrome user-data directory must be an absolute path")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return chromeProfile{}, fmt.Errorf("Chrome user-data directory is unavailable: %w", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return chromeProfile{}, fmt.Errorf("read Chrome user-data directory: %w", err)
	}
	if requested != "" && (requested != filepath.Base(requested) || requested == "." || requested == "..") {
		return chromeProfile{}, fmt.Errorf("invalid Chrome profile name")
	}
	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name != "Default" && !strings.HasPrefix(name, "Profile ") {
			continue
		}
		if requested != "" && name != requested {
			continue
		}
		has, err := profileHasAgent(filepath.Join(root, name))
		if err != nil {
			return chromeProfile{}, err
		}
		if has {
			profiles = append(profiles, name)
		}
	}
	sort.Strings(profiles)
	if len(profiles) == 0 {
		return chromeProfile{}, fmt.Errorf("no Chrome profile with an enabled Browser Agent extension was found; check its installation and profile")
	}
	if len(profiles) > 1 {
		return chromeProfile{}, fmt.Errorf("Browser Agent is installed in multiple Chrome profiles (%s); set BROWSER_AGENT_CHROME_PROFILE to choose one", strings.Join(profiles, ", "))
	}
	return chromeProfile{Root: root, Name: profiles[0]}, nil
}
