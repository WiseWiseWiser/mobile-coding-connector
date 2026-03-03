# Plan: Checkpoint-Based Filesystem with Mobile Diff Viewing

## Overview

Add a **checkpoint** system to the Files tab that captures snapshots of changed files at specific points in time. Users can view diffs between consecutive checkpoints, optimized for mobile screens.

## Core Concepts

### Checkpoint
A checkpoint is a named snapshot taken at a point in time. It records:
- **ID**: Auto-incrementing integer (1, 2, 3, ...)
- **Name**: Auto-generated or user-provided (e.g., "Checkpoint #1", "Added port forwarding")
- **Timestamp**: When the checkpoint was created
- **Files**: Map of file paths to their snapshot state

### File Snapshot
Each file in a checkpoint stores:
- **Path**: Relative file path (e.g., `server/portforward/portforward.go`)
- **Status**: `added` | `modified` | `deleted`
- **Content**: Full file content at this point (empty string for deleted files)

### Base (Checkpoint 0)
The implicit "base" checkpoint is the current git HEAD state. The first user-created checkpoint diffs against HEAD. Subsequent checkpoints diff against the previous checkpoint.

## Architecture

### Backend

#### Data Model (`server/checkpoint/`)

```go
type FileSnapshot struct {
    Path    string `json:"path"`
    Status  string `json:"status"`  // "added", "modified", "deleted"
    Content string `json:"content"` // full content (empty for deleted)
}

type Checkpoint struct {
    ID        int             `json:"id"`
    Name      string          `json:"name"`
    Timestamp time.Time       `json:"timestamp"`
    Files     []FileSnapshot  `json:"files"`
}
```

#### Storage
- Checkpoints stored in-memory (with optional JSON file persistence at `.checkpoints.json`)
- Base content retrieved via `git show HEAD:<path>` on demand

#### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/checkpoints` | List all checkpoints (id, name, timestamp, file count) |
| `POST` | `/api/checkpoints` | Create a new checkpoint from current working tree changes |
| `GET` | `/api/checkpoints/:id` | Get checkpoint details (list of changed files with status) |
| `DELETE` | `/api/checkpoints/:id` | Delete a checkpoint |
| `GET` | `/api/checkpoints/:id/diff` | Get diff between this checkpoint and the previous one |
| `GET` | `/api/checkpoints/:id/diff/:file` | Get file-level diff (unified diff format) |
| `GET` | `/api/checkpoints/current` | Get current working tree changes vs. last checkpoint |

#### Creating a Checkpoint
1. Run `git diff --name-status HEAD` (or vs. last checkpoint) to find changed files
2. For each changed file, read its full content from disk
3. Store as a new `Checkpoint` with all file snapshots
4. Return the checkpoint ID

#### Computing Diffs
- **Checkpoint N vs N-1**: Compare file contents stored in checkpoint N against checkpoint N-1
- **Checkpoint 1 vs base**: Compare checkpoint 1 file contents against `git show HEAD:<path>`
- Use Go's `github.com/sergi/go-diff/diffmatchpatch` or generate unified diff format
- Return diff in a structured format suitable for rendering:

```go
type FileDiff struct {
    Path      string     `json:"path"`
    Status    string     `json:"status"`
    Hunks     []DiffHunk `json:"hunks"`
}

type DiffHunk struct {
    OldStart int        `json:"old_start"`
    OldLines int        `json:"old_lines"`
    NewStart int        `json:"new_start"`
    NewLines int        `json:"new_lines"`
    Lines    []DiffLine `json:"lines"`
}

type DiffLine struct {
    Type    string `json:"type"`  // "context", "add", "delete"
    Content string `json:"content"`
    OldNum  int    `json:"old_num,omitempty"`
    NewNum  int    `json:"new_num,omitempty"`
}
```

### Frontend

#### Mobile-Optimized Diff Viewing

The diff viewer must work well on narrow mobile screens (320-428px width). Key design decisions:

##### 1. Unified Diff (not side-by-side)
- Side-by-side is unusable on mobile (each side would be ~160px)
- Use unified diff: deletions in red, additions in green, context in neutral
- Single column, full width

##### 2. File List → File Diff Drill-down
- **Checkpoint list page**: Shows list of checkpoints as cards with timestamp and file count
- **Checkpoint detail page**: Shows list of changed files with status badges (A/M/D)
- **File diff page**: Shows the actual unified diff for a single file
- Navigation via URL: `/v2?tab=files&view=checkpoint-3&file=server/portforward/portforward.go`

##### 3. Diff Line Rendering
```
.diff-line-delete { background: rgba(239, 68, 68, 0.15); color: #fca5a5; }
.diff-line-add    { background: rgba(34, 197, 94, 0.15); color: #86efac; }
.diff-line-ctx    { color: #64748b; }
```
- Line numbers shown in a narrow gutter (40px)
- Code uses monospace font, horizontal scroll on overflow
- Hunk headers shown as separators: `@@ -10,5 +10,7 @@`

##### 4. Collapsed Hunks
- By default, only changed hunks are expanded
- Large unchanged regions between hunks show "... N lines hidden ..." that can be tapped to expand
- This keeps mobile scrolling manageable

##### 5. Swipe Navigation
- Swipe left/right to navigate between files in the same checkpoint
- File name shown in sticky header at top

#### Component Structure

```
FilesView
├── CheckpointListView          # List of all checkpoints
│   ├── CreateCheckpointButton  # Floating action button
│   └── CheckpointCard[]        # Each checkpoint summary
├── CheckpointDetailView        # Files in a single checkpoint
│   └── ChangedFileRow[]        # File path + status badge
├── FileDiffView                # Unified diff for one file
│   ├── DiffHunk[]              # Expandable diff hunks
│   │   └── DiffLine[]          # Individual diff lines
│   └── FileNavigator           # Prev/Next file buttons
└── CurrentChangesView          # Live working tree vs last checkpoint
```

#### URL Routing (via search params)

| URL Params | View |
|------------|------|
| `tab=files` | Checkpoint list |
| `tab=files&view=checkpoint-3` | Checkpoint #3 detail (file list) |
| `tab=files&view=checkpoint-3&file=path/to/file.go` | File diff |
| `tab=files&view=current` | Current working tree changes |

#### React Hook: `useCheckpoints`

```typescript
interface Checkpoint {
    id: number;
    name: string;
    timestamp: string;
    fileCount: number;
}

interface CheckpointDetail extends Checkpoint {
    files: { path: string; status: 'added' | 'modified' | 'deleted' }[];
}

interface FileDiff {
    path: string;
    status: string;
    hunks: DiffHunk[];
}

function useCheckpoints() {
    return {
        checkpoints: Checkpoint[],
        loading: boolean,
        createCheckpoint: (name?: string) => Promise<void>,
        deleteCheckpoint: (id: number) => Promise<void>,
        getDetail: (id: number) => Promise<CheckpointDetail>,
        getFileDiff: (id: number, file: string) => Promise<FileDiff>,
        getCurrentChanges: () => Promise<FileDiff[]>,
        refresh: () => void,
    };
}
```

## Implementation Order

### Phase 1: Backend Core
1. Create `server/checkpoint/` package with data model
2. Implement checkpoint creation from git working tree
3. Implement diff computation between checkpoints
4. Register API endpoints in `server/server.go`

### Phase 2: Frontend - Checkpoint List
1. Implement `useCheckpoints` hook
2. Replace mock `FilesView` with `CheckpointListView`
3. Add "Create Checkpoint" button
4. URL routing for `view` param within files tab

### Phase 3: Frontend - Diff Viewer
1. Implement `CheckpointDetailView` (file list for a checkpoint)
2. Implement `FileDiffView` with unified diff rendering
3. Mobile-optimized diff styling (colors, line numbers, monospace)
4. Collapsible hunks for large diffs

### Phase 4: Polish
1. Swipe navigation between files
2. "Current changes" view (live working tree vs last checkpoint)
3. Checkpoint naming/renaming
4. Persistence to disk (`.checkpoints.json`)

## Mobile UX Considerations

- **Touch targets**: All tappable areas at least 44px tall
- **Scroll performance**: Virtualize long diffs (only render visible hunks)
- **Font size**: Code at 12px monospace, minimum readable on mobile
- **Horizontal overflow**: Code lines scroll horizontally independently (not the whole page)
- **Dark theme**: Consistent with existing `.mcc-` dark theme
- **Loading states**: Skeleton loaders for checkpoint list and diff views
- **Sticky headers**: File name sticks to top while scrolling through diff
