package machinebackup

import "time"

const manifestVersion = 1

// BackupRequest is the JSON body for POST /api/remote-agent/machine/backup.
type BackupRequest struct {
	DryRun                 bool     `json:"dry_run"`
	Exclude                []string `json:"exclude"`
	Include                []string `json:"include"`
	LargeDirThresholdBytes int64    `json:"large_dir_threshold_bytes,omitempty"`
	SkipGitDirsScan        bool     `json:"skip_git_dirs_scan,omitempty"`
	GitDirsScanMaxDepth    int      `json:"git_dirs_scan_max_depth,omitempty"` // 0 = unlimited
}

// BackupStreamRequest is the JSON body for POST /api/remote-agent/machine/backup/stream.
type BackupStreamRequest struct {
	Exclude                []string `json:"exclude"`
	Include                []string `json:"include"`
	LargeDirThresholdBytes int64    `json:"large_dir_threshold_bytes,omitempty"`
	SkipGitDirsScan        bool     `json:"skip_git_dirs_scan,omitempty"`
	GitDirsScanMaxDepth    int      `json:"git_dirs_scan_max_depth,omitempty"` // 0 = unlimited
}

// GitScanOptions configures git repo discovery during backup.
type GitScanOptions struct {
	SkipGitDirsScan     bool
	GitDirsScanMaxDepth int // 0 = unlimited
}

// BackupConfigRequest is the JSON body for PUT /api/remote-agent/machine/backup-config.
type BackupConfigRequest struct {
	Exclude           []string `json:"exclude"`
	LargeDirThreshold string   `json:"large_dir_threshold,omitempty"`
}

// FileStat is one dot-file entry with byte size.
type FileStat struct {
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	Symlink bool   `json:"symlink,omitempty"`
}

// SectionTotals rolls up counts and bytes for a backup section.
type SectionTotals struct {
	Files    int   `json:"files"`
	Symlinks int   `json:"symlinks"`
	Bytes    int64 `json:"bytes"`
}

// DirStat summarizes one included dot-directory at the home root.
type DirStat struct {
	Path     string `json:"path"`
	Files    int    `json:"files"`
	Dirs     int    `json:"dirs"`
	Symlinks int    `json:"symlinks"`
	Bytes    int64  `json:"bytes"`
}

// Manifest is written as manifest.json inside the archive.
type Manifest struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Home      string    `json:"home"`
	Excluded  []string  `json:"excluded"`
	DirStats  []DirStat `json:"dir_stats"`
	DotFiles  []string  `json:"dot_files"`
}

// GitRepoWorktreesSnapshot is written to .backup/git-repo-worktrees.json.
type GitRepoWorktreesSnapshot struct {
	Version    string         `json:"version"`
	CapturedAt time.Time      `json:"captured_at"`
	Repos      []GitRepoEntry `json:"repos"`
}

// GitRepoEntry is one main repository with optional linked worktrees.
type GitRepoEntry struct {
	Path      string             `json:"path"`
	Branch    string             `json:"branch,omitempty"`
	CommitSHA string             `json:"commit_sha,omitempty"`
	CommitMsg string             `json:"commit_msg,omitempty"`
	Status    string             `json:"status,omitempty"`
	Error     string             `json:"error,omitempty"`
	Worktrees []GitWorktreeEntry `json:"worktrees,omitempty"`
}

// GitWorktreeEntry is one linked worktree checkout.
type GitWorktreeEntry struct {
	Path      string `json:"path"`
	Branch    string `json:"branch,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
	CommitMsg string `json:"commit_msg,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

// MachineBackupPlan is returned when dry_run is true or in a stream done frame.
type MachineBackupPlan struct {
	Home          string        `json:"home"`
	DotFiles      []FileStat    `json:"dot_files"`
	AllFiles      []FileStat    `json:"all_files"`
	DotFilesTotal SectionTotals `json:"dot_files_total"`
	DirStats      []DirStat     `json:"dir_stats"`
	DotDirsTotal  SectionTotals `json:"dot_dirs_total"`
	GrandTotal    SectionTotals `json:"grand_total"`
	Excluded      []ExcludePathEntry `json:"excluded"`
	Included      []string           `json:"included"`
	GitRepos      *GitRepoWorktreesSnapshot `json:"git_repos,omitempty"`
}

// RestoreEntry describes one restore action.
type RestoreEntry struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// MachineRestorePlan is returned for restore dry-run and apply (JSON endpoint).
type MachineRestorePlan struct {
	Home    string         `json:"home"`
	Entries []RestoreEntry `json:"entries"`
}

// MachineRestoreSummary is the restore stream done payload.
type MachineRestoreSummary struct {
	Home          string `json:"home"`
	SkipIdentical int    `json:"skip_identical"`
	Update        int    `json:"update"`
	Create        int    `json:"create"`
	TotalEntries  int    `json:"total_entries"`
}

// archiveMember is one path collected for backup.
type archiveMember struct {
	RelPath   string
	Mode      int64
	Linkname  string
	IsDir     bool
	IsSymlink bool
}