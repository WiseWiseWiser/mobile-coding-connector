package diskmon

// Cleanup groups mirror reclaim's registry groups; the web panel renders one
// section per group in this order.
const (
	cleanupGroupDefault    = "default"
	cleanupGroupAggressive = "aggressive"
	cleanupGroupSystem     = "system"
	cleanupGroupInventory  = "inventory"
)

// Action kinds drive the badge next to a rendered command.
const (
	// cleanupActionRemove renders "rm -rf": the engine deletes the paths
	// itself, the shell line is only the equivalent a user could paste.
	cleanupActionRemove = "rm-rf"
	// cleanupActionOfficial delegates to a tool (go clean -cache, brew cleanup).
	cleanupActionOfficial = "official"
	// cleanupActionSudo needs root and is rendered with a sudo badge.
	cleanupActionSudo = "sudo"
	// cleanupActionNone only describes a row (inventory / never cleared).
	cleanupActionNone = "none"
)

// CleanupAction is one rendered command line of a cleanup class.
type CleanupAction struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// CleanupClass is one selectable (or inventory-only) cleanup class. The command
// text comes from the engine's op projection, so it always matches what Apply
// actually removes.
type CleanupClass struct {
	ID         string          `json:"id"`
	Group      string          `json:"group"`
	Label      string          `json:"label"`
	SizeBytes  *int64          `json:"sizeBytes"`
	Measured   bool            `json:"measured"`
	Selected   bool            `json:"selected"`
	NeedsAdmin bool            `json:"needsAdmin,omitempty"`
	ReadOnly   bool            `json:"readOnly,omitempty"`
	Actions    []CleanupAction `json:"actions"`
	Command    string          `json:"command"`
	Paths      []string        `json:"paths,omitempty"`
	Notes      []string        `json:"notes,omitempty"`
	Details    []string        `json:"details,omitempty"`
}

// CleanupGroup is a section in the clear panel.
type CleanupGroup struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note,omitempty"`
}

// CleanupVolume is the data volume holding Home.
type CleanupVolume struct {
	Path       string `json:"path"`
	TotalBytes int64  `json:"totalBytes"`
	FreeBytes  int64  `json:"freeBytes"`
}

// CleanupTotals are the sums the panel shows above the class list.
type CleanupTotals struct {
	SelectedBytes   int64 `json:"selectedBytes"`
	AggressiveBytes int64 `json:"aggressiveBytes"`
	SystemBytes     int64 `json:"systemBytes"`
	InventoryBytes  int64 `json:"inventoryBytes"`
	Unmeasured      int   `json:"unmeasured"`
}

// CleanupPlan is GET /api/server/disk/cleanup/plan.
type CleanupPlan struct {
	ScannedAt  string `json:"scannedAt"`
	ScanMillis int64  `json:"scanMillis"`
	// Pending is true while a measurement pass is still running: the sizes in
	// this response are from the previous pass, or missing on the first call.
	Pending bool           `json:"pending,omitempty"`
	Home    string         `json:"home"`
	Volume  *CleanupVolume `json:"volume,omitempty"`
	Groups  []CleanupGroup `json:"groups"`
	Classes []CleanupClass `json:"classes"`
	Totals  CleanupTotals  `json:"totals"`
}

// CleanupSkip is a class the server refused to run, with the reason and the
// copyable command so the user can still free the space by hand.
type CleanupSkip struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Reason  string `json:"reason"`
	Command string `json:"command"`
}

// CleanupStepResult is one finished step of a run.
type CleanupStepResult struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Freed int64  `json:"freed"`
	Error string `json:"error,omitempty"`
}

// CleanupRunResult is the done frame's payload.
type CleanupRunResult struct {
	FreedBytes int64               `json:"freedBytes"`
	Steps      []CleanupStepResult `json:"steps"`
	Skipped    []CleanupSkip       `json:"skipped,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// Sudo policies for a run.
const (
	// cleanupSudoAsk may pop the macOS authorization dialog for system classes.
	cleanupSudoAsk = "ask"
	// cleanupSudoNever runs unprivileged only and skips system classes.
	cleanupSudoNever = "never"
)
