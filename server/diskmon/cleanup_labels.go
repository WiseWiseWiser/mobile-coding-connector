package diskmon

import (
	"path/filepath"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/disk/reclaim"
)

// cleanupDisplay is the presentation-only side of a cleanup class: the shared
// reclaim registry owns what a step does, the host owns how it reads.
type cleanupDisplay struct {
	label string
	notes []string
}

// cleanupDisplays mirrors the engine's step list in English. The keys are the
// reclaim step IDs; a missing entry falls back to the engine summary.
var cleanupDisplays = map[string]cleanupDisplay{
	// default: always selected
	"sandbox-gocache": {label: "Sandbox Go cache", notes: []string{"Build cache for sandboxed builds; regenerated on demand."}},
	"codex-logs":      {label: "Codex log databases"},
	"home-logs":       {label: "ai-critic server logs"},
	"brew-cleanup":    {label: "Homebrew downloads", notes: []string{"Runs the official cleanup, then removes the download cache it leaves behind."}},
	"npm-cache":       {label: "npm cache", notes: []string{"Official command; the cache rebuilds on the next install."}},
	"remote-agent-logs": {
		label: "remote-agent logs",
	},
	"podman-applehv-cache": {label: "Podman VM image cache", notes: []string{"Image layers are re-pulled on the next run."}},
	"iterm-bak-apps":       {label: "iTerm backup bundles", notes: []string{"Skips bundles belonging to a running iTerm."}},

	// aggressive: opt-in, costs a rebuild or re-download
	"go-build-cache":    {label: "Go build cache", notes: []string{"Pure compiled output; the Go module cache is never touched."}},
	"pnpm-store":        {label: "pnpm store", notes: []string{"Prunes unreferenced packages; installed projects keep working."}},
	"npm-npx":           {label: "npm _npx", notes: []string{"One-shot npx packages, re-downloaded when needed."}},
	"bun-cache":         {label: "Bun cache"},
	"codex-releases":    {label: "Old codex runtimes", notes: []string{"Keeps the newest runtime and deletes the rest."}},
	"xcache":            {label: "~/.cache", notes: []string{"Whole directory: only re-download cost, no project data."}},
	"ms-playwright":     {label: "Playwright browsers", notes: []string{"Browsers are re-downloaded by `playwright install`."}},
	"opencode-snapshot": {label: "OpenCode snapshots"},

	// system: admin required, macOS may still refuse
	"ios-sim-runtimes":    {label: "iOS Simulator runtimes", notes: []string{"Runs `xcrun simctl runtime delete all` under the hood; skipped when nothing is installed."}},
	"ios-sim-mobileasset": {label: "iOS mobile assets", notes: []string{"SIP may refuse the delete even as root; the command stays copyable."}},
	"dirs-cleaner":        {label: "dirs_cleaner leftovers", notes: []string{"Files macOS moved aside at boot; needs admin authorization."}},

	// inventory: listed, never cleared
	"shipit":       {label: "ShipIt updates", notes: []string{"Read-only: update the app to reclaim this space."}},
	"go-mod-cache": {label: "Go module cache", notes: []string{"Accounting only — deleting it forces a network re-fetch of every module."}},
}

func cleanupLabel(info reclaim.StepInfo) string {
	if d, ok := cleanupDisplays[info.ID]; ok && d.label != "" {
		return d.label
	}
	return info.Summary
}

func cleanupNotes(info reclaim.StepInfo) []string {
	if d, ok := cleanupDisplays[info.ID]; ok {
		return d.notes
	}
	return nil
}

// cleanupCommands renders the step's ops as display rows. Every command shown to
// the user comes from here, never from a user-typed string, and never by
// joining anything into a shell.
func cleanupCommands(home string, info reclaim.StepInfo) []CleanupAction {
	actions := make([]CleanupAction, 0, len(info.Actions))
	for _, op := range info.Actions {
		kind := cleanupActionOfficial
		switch {
		case op.Kind == reclaim.OpNone:
			kind = cleanupActionNone
		case op.Sudo:
			kind = cleanupActionSudo
		case op.Kind == reclaim.OpRemoveDir || op.Kind == reclaim.OpRemoveFiles:
			kind = cleanupActionRemove
		}
		actions = append(actions, CleanupAction{Kind: kind, Text: cleanupActionText(home, info.ID, op, kind)})
	}
	return actions
}

// cleanupHeadline is the full command of the first non-hidden op, kept for
// copy/diagnostics. It is the same text the class row shows, with the `rm -rf`
// prefix included.
func cleanupHeadline(home string, info reclaim.StepInfo) string {
	for _, op := range info.Actions {
		if op.Hidden {
			continue
		}
		return cleanupFullCommand(home, info.ID, op)
	}
	if len(info.Actions) > 0 {
		return cleanupFullCommand(home, info.ID, info.Actions[0])
	}
	return ""
}

// cleanupActionText is the row text: the target path for plain deletes (the
// badge already says `rm -rf`), the full line otherwise.
func cleanupActionText(home string, id string, op reclaim.Op, kind string) string {
	full := cleanupFullCommand(home, id, op)
	if kind == cleanupActionRemove {
		return strings.TrimPrefix(strings.TrimPrefix(full, "rm -rf "), "rm ")
	}
	return full
}

// cleanupFullCommand renders one op as the command it is equivalent to.
func cleanupFullCommand(home string, id string, op reclaim.Op) string {
	if fn, ok := cleanupCommandOverrides[id]; ok {
		return fn(home, op)
	}
	switch op.Kind {
	case reclaim.OpRemoveDir:
		return "rm -rf " + cleanupShortPath(home, firstPath(op.Paths))
	case reclaim.OpRemoveFiles:
		return "rm " + cleanupShortPath(home, firstPath(op.Paths))
	case reclaim.OpExec:
		if len(op.Argv) == 0 {
			// Some engine ops carry a whole pre-rendered command in Text
			// (dirs-cleaner: `sudo find … -exec rm -rf {} +`).
			if op.Text != "" {
				return cleanupShortPath(home, op.Text)
			}
			return "—"
		}
		line := strings.Join(op.Argv, " ")
		if op.Sudo {
			line = "sudo " + line
		}
		return line
	default:
		return "—"
	}
}

// cleanupCommandOverrides covers the classes whose command is not a plain
// delete of Paths (partial deletes, two-step root deletes, tool commands that
// only measure via Paths).
var cleanupCommandOverrides = map[string]func(home string, op reclaim.Op) string{
	"codex-releases": func(home string, op reclaim.Op) string {
		return "rm -rf " + cleanupShortPath(home, firstPath(op.Paths)) + "/<all but newest>"
	},
	"ios-sim-mobileasset": func(home string, op reclaim.Op) string {
		p := cleanupShortPath(home, firstPath(op.Paths))
		return "sudo chflags -R norestricted -- " + p + " && sudo rm -rf -- " + p
	},
	"shipit":       func(string, reclaim.Op) string { return "—" },
	"go-mod-cache": func(string, reclaim.Op) string { return "go clean -modcache" },
}

// cleanupShortPath abbreviates a path under HOME as ~/… for readability; the
// absolute path stays available in CleanupClass.Paths.
func cleanupShortPath(home string, p string) string {
	if home == "" || p == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + rel
	}
	return p
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}
