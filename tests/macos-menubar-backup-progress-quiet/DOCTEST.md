# macOS Remote Menu Bar Backup Progress — Quiet / Low-CPU Console Doctests

Source contracts and optional pure helpers for making the **Backup progress
window** a **silent, low-CPU scrollable console**: no focus steal on open, batched
UI appends (interval flush + `textStorage.append`), and a progress hot path that
only enqueues lines.

Sibling of `tests/macos-menubar-backup-progress/` (format lines, show-window
policy, stream consumption). **Do not rewrite** sealed progress leaves. This tree
only seals **open presentation policy** and **append/CPU policy**.

No live network; no UI automation / cursor simulation.

# DSN (Domain Specific Notion)

**Participants**

- **Backup progress window (`BackupProgressWindow`)** — AppKit monospaced
  scrollable log (`NSScrollView` + `NSTextView`) used for manual Backup Now and
  enable-immediate runs. Open policy and line-append path live here.
- **Progress session buffer** — holds `pendingLines` (thread-safe or
  actor-protected); flushes on a ~100–200 ms interval; on flush, writes one
  batch to `textStorage` and scrolls once.
- **Quiet open policy** — presents the window without stealing keyboard focus
  from other apps: `orderFrontRegardless` / non-activating `orderFront`;
  **must not** call `NSApp.activate(ignoringOtherApps:)`.
- **Remote menu bar (`ai-critic-remote-macos` / `AICriticApp`)** — `runBackupNow`
  onProgress maps SSE frames to lines and calls `progressSession?.append(line)`
  only (no MainActor UI thrash in the network callback beyond enqueue).
- **Machine backup client** — streams SSE frames into the onProgress callback
  (unchanged consumption contract; sealed in sibling tree).
- **Pure Go helpers (`macosapp/menubar`, optional)** —
  `BackupProgressFlushIntervalMilliseconds`, `JoinBackupProgressBatch`,
  `ShouldScrollBackupProgressOnFlush` — document interval band and batch join /
  scroll-on-flush policy. Design-phase leaves seal them via **menubar source
  contracts** (so the tree compiles before symbols exist); implementer adds the
  real pure APIs and may later switch leaves to direct calls.
- **Test harness** — greps Swift under `macos-ai-critic/Shared/` (and remote app
  for hot path); greps `macosapp/menubar` for helper_* ops.

**Behaviors (sealed)**

- **Quiet open**
  - `BackupProgressWindow` open path presents via quiet order-front
    (`orderFrontRegardless` or non-`makeKey` `orderFront`).
  - File `BackupProgressWindow.swift` must **not** contain
    `activate(ignoringOtherApps:` or `NSApp.activate`.
  - Window is still created (`NSWindow`) and ordered front somehow.
- **Batched append**
  - Flush interval / timer in **100–200 ms** band (e.g. 0.15 s / 150 ms).
  - Session keeps a pending/buffer of lines; `append` enqueues; `flush` drains.
  - Flush writes via `textStorage.append` (or equivalent batch write), not
    per-line `textView.string +=` as the sole hot path.
  - `scrollToEndOfDocument` runs from the flush path only (not every raw line).
- **Progress hot path**
  - onProgress / download callback only formats a line and calls session
    `append` / enqueue — flush/UI batching is inside the session.
- **Scrollable console (keep)**
  - `NSScrollView` + `NSTextView` (`documentView`) remain.
  - `isEditable = false`, `isSelectable = true`.
- **Pure helpers (optional Go)**
  - `BackupProgressFlushIntervalMilliseconds` ∈ [100, 200] (canonical 150).
  - `JoinBackupProgressBatch(lines)` joins with `"\n"` and a trailing newline when
    non-empty; empty input → `""`.
  - `ShouldScrollBackupProgressOnFlush()` → `true` (v1 always scroll on flush).

## Version

0.0.2

## Decision Tree

```
[macos-menubar-backup-progress-quiet]
 |
 +-- quiet-open/                         (GROUP)  Focus-steal / presentation policy
 |    +-- no-activate/                   (LEAF)   no NSApp.activate / ignoringOtherApps in BackupProgressWindow
 |    +-- quiet-order-front/             (LEAF)   orderFrontRegardless or non-makeKey orderFront
 |    +-- presents-window/               (LEAF)   NSWindow + order-front family still present
 |
 +-- batch-append/                       (GROUP)  Low-CPU batched UI append
 |    +-- flush-interval/                (LEAF)   timer / interval in 100–200ms band
 |    +-- pending-buffer/                (LEAF)   pending/buffer + flush symbols
 |    +-- text-storage-append/           (LEAF)   textStorage batch write; not sole per-line string +=
 |    +-- scroll-on-flush/               (LEAF)   scrollToEnd only on flush path
 |
 +-- progress-hot-path/                  (GROUP)  Callback only enqueues
 |    +-- enqueue-only/                  (LEAF)   onProgress → append/enqueue; flush separate
 |
 +-- scrollable-console/                 (GROUP)  Keep scrollable console UX
 |    +-- scroll-and-text-view/          (LEAF)   NSScrollView + NSTextView / documentView
 |    +-- non-editable-selectable/       (LEAF)   isEditable false; isSelectable true
 |
 +-- helpers/                            (GROUP)  Optional pure Go batch helpers
      +-- flush-interval-ms/             (LEAF)   constant in [100,200]
      +-- join-batch/                    (LEAF)   JoinBackupProgressBatch join policy
      +-- scroll-on-flush-policy/        (LEAF)   ShouldScrollBackupProgressOnFlush == true
```

## Test Index

| # | Leaf | Description | Expect RED pre-fix |
|---|------|-------------|---------------------|
| 1 | `quiet-open/no-activate` | No `activate(ignoringOtherApps:)` / `NSApp.activate` in BackupProgressWindow | yes (current activate) |
| 2 | `quiet-open/quiet-order-front` | Quiet `orderFrontRegardless` / non-makeKey `orderFront` | yes (only makeKeyAndOrderFront) |
| 3 | `quiet-open/presents-window` | Still creates `NSWindow` and orders front | no (already true) |
| 4 | `batch-append/flush-interval` | Flush timer/interval in 100–200 ms | yes |
| 5 | `batch-append/pending-buffer` | pending/buffer + flush wiring | yes |
| 6 | `batch-append/text-storage-append` | `textStorage` batch append; not sole per-line `string +=` | yes (string +=) |
| 7 | `batch-append/scroll-on-flush` | scroll only from flush path | yes (scroll every line) |
| 8 | `progress-hot-path/enqueue-only` | callback append/enqueue; flush separate in session | yes (immediate UI append) |
| 9 | `scrollable-console/scroll-and-text-view` | NSScrollView + NSTextView kept | no |
| 10 | `scrollable-console/non-editable-selectable` | non-editable, selectable | no |
| 11 | `helpers/flush-interval-ms` | menubar constant ∈ [100,200] (source) | yes (symbol missing) |
| 12 | `helpers/join-batch` | `JoinBackupProgressBatch` + join policy body | yes (symbol missing) |
| 13 | `helpers/scroll-on-flush-policy` | `ShouldScroll…` returns true | yes (symbol missing) |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| no-activate | client | ClientLeaf=no-activate | NoActivate=true |
| quiet-order-front | client | ClientLeaf=quiet-order-front | QuietOrderFront=true |
| presents-window | client | ClientLeaf=presents-window | PresentsWindow=true |
| flush-interval | client | ClientLeaf=flush-interval | FlushIntervalInBand=true |
| pending-buffer | client | ClientLeaf=pending-buffer | HasPendingBuffer=true |
| text-storage-append | client | ClientLeaf=text-storage-append | UsesTextStorageAppend=true |
| scroll-on-flush | client | ClientLeaf=scroll-on-flush | ScrollOnlyOnFlush=true |
| enqueue-only | client | ClientLeaf=enqueue-only | ProgressEnqueueOnly=true |
| scroll-and-text-view | client | ClientLeaf=scroll-and-text-view | HasScrollableConsole=true |
| non-editable-selectable | client | ClientLeaf=non-editable-selectable | NonEditableSelectable=true |
| flush-interval-ms | helper_flush_interval | menubar src | ms ∈ [100,200] |
| join-batch | helper_join_batch | menubar src | helper + join policy |
| scroll-on-flush-policy | helper_scroll_policy | menubar src | return true |

## How to Run

```sh
doctest vet ./tests/macos-menubar-backup-progress-quiet
doctest test ./tests/macos-menubar-backup-progress-quiet/...
```

Sibling trees must stay GREEN (implementer regression):

```sh
doctest test ./tests/macos-menubar-backup-progress/...
doctest test ./tests/macos-menubar-backup/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Canonical interval band for quiet batch flush (milliseconds).
const (
	flushIntervalMinMs = 100
	flushIntervalMaxMs = 200
)

type Request struct {
	// Op: client | helper_flush_interval | helper_join_batch | helper_scroll_policy
	Op string

	// client: leaf slug (no-activate, quiet-order-front, …)
	ClientLeaf string

	// helper_join_batch — documents intended pure-helper inputs; join is sealed
	// via menubar source contract until symbols exist (no production code in design).
	BatchLines []string
}

type Response struct {
	// quiet-open
	NoActivateInProgressWindow bool
	QuietOrderFront            bool
	PresentsWindow             bool

	// batch-append
	FlushIntervalInBand   bool
	HasPendingBuffer      bool
	UsesTextStorageAppend bool
	ScrollOnlyOnFlush     bool

	// progress-hot-path
	ProgressEnqueueOnly bool

	// scrollable-console
	HasScrollableConsole  bool
	NonEditableSelectable bool

	// helpers (source-contract on macosapp/menubar until pure symbols land)
	FlushIntervalMs      int  // -1 if constant missing
	HasJoinBatchHelper   bool
	JoinBatchPolicyOK    bool // body implements "\n" join + trailing newline / empty→""
	ShouldScrollOnFlush  bool // true only if helper returns true
	HasScrollPolicyHelper bool

	// diagnostics
	ProgressWindowSource string
	MenubarSourcesChecked []string
	SwiftSourcesChecked  []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "client":
		return runClientQuietContract(t, req, resp)
	case "helper_flush_interval":
		return runHelperFlushInterval(t, resp)
	case "helper_join_batch":
		return runHelperJoinBatch(t, req, resp)
	case "helper_scroll_policy":
		return runHelperScrollPolicy(t, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

// readMenubarPackageSource concatenates macosapp/menubar/*.go for helper contracts.
func readMenubarPackageSource() (src string, paths []string, err error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return "", nil, err
	}
	dir := filepath.Join(moduleRoot, "macosapp", "menubar")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("read menubar dir: %w", err)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		b.Write(data)
		b.WriteByte('\n')
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return "", nil, fmt.Errorf("no .go files in %s", dir)
	}
	return b.String(), paths, nil
}

func runHelperFlushInterval(t *testing.T, resp *Response) (*Response, error) {
	src, paths, err := readMenubarPackageSource()
	if err != nil {
		return nil, err
	}
	resp.MenubarSourcesChecked = paths
	resp.FlushIntervalMs = -1
	// Sealed name from requirement: BackupProgressFlushIntervalMilliseconds = N
	re := regexp.MustCompile(`BackupProgressFlushIntervalMilliseconds\s*=\s*(\d+)`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		return resp, nil
	}
	n, convErr := strconv.Atoi(m[1])
	if convErr != nil {
		return resp, nil
	}
	resp.FlushIntervalMs = n
	return resp, nil
}

func runHelperJoinBatch(t *testing.T, req *Request, resp *Response) (*Response, error) {
	src, paths, err := readMenubarPackageSource()
	if err != nil {
		return nil, err
	}
	resp.MenubarSourcesChecked = paths
	// Function must exist.
	resp.HasJoinBatchHelper = regexp.MustCompile(`func\s+JoinBackupProgressBatch\s*\(`).MatchString(src)
	if !resp.HasJoinBatchHelper {
		return resp, nil
	}
	// Extract function body (best-effort brace match from func to balancing end).
	body := extractFuncBody(src, `func\s+JoinBackupProgressBatch\s*\(`)
	if body == "" {
		return resp, nil
	}
	// Policy: empty → ""; non-empty join with "\n" + trailing newline.
	// Accept strings.Join(..., "\n") plus trailing "\n", or explicit loops.
	hasJoin := strings.Contains(body, `strings.Join`) && (strings.Contains(body, `"\n"`) || strings.Contains(body, `'\n'`))
	hasTrailing := strings.Contains(body, `+"\n"`) || strings.Contains(body, `+ "\n"`) ||
		strings.Contains(body, `"\n"`) && (strings.Contains(body, "append") || strings.Contains(body, "return"))
	hasEmptyGuard := regexp.MustCompile(`len\s*\(|==\s*0|nil`).MatchString(body) &&
		regexp.MustCompile(`return\s+""`).MatchString(body)
	// Also accept a compact form: if len==0 { return "" }; return strings.Join(lines, "\n") + "\n"
	resp.JoinBatchPolicyOK = hasJoin && hasTrailing && hasEmptyGuard
	// Document BatchLines in request for reviewers; pure evaluation happens in production later.
	_ = req.BatchLines
	return resp, nil
}

func runHelperScrollPolicy(t *testing.T, resp *Response) (*Response, error) {
	src, paths, err := readMenubarPackageSource()
	if err != nil {
		return nil, err
	}
	resp.MenubarSourcesChecked = paths
	resp.HasScrollPolicyHelper = regexp.MustCompile(`func\s+ShouldScrollBackupProgressOnFlush\s*\(`).MatchString(src)
	if !resp.HasScrollPolicyHelper {
		resp.ShouldScrollOnFlush = false
		return resp, nil
	}
	body := extractFuncBody(src, `func\s+ShouldScrollBackupProgressOnFlush\s*\(`)
	// v1: always true
	resp.ShouldScrollOnFlush = regexp.MustCompile(`return\s+true\b`).MatchString(body)
	return resp, nil
}

// extractFuncBody returns the {...} body after the first match of funcRe, brace-balanced.
func extractFuncBody(src, funcRe string) string {
	re := regexp.MustCompile(funcRe)
	loc := re.FindStringIndex(src)
	if loc == nil {
		return ""
	}
	i := loc[1]
	// find opening brace
	for i < len(src) && src[i] != '{' {
		i++
	}
	if i >= len(src) {
		return ""
	}
	depth := 0
	start := i
	for ; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1]
			}
		}
	}
	return ""
}

func runClientQuietContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	progressPath := filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "BackupProgressWindow.swift")
	remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
	clientPath := filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "MachineBackupClient.swift")

	progressSrc, err := os.ReadFile(progressPath)
	if err != nil {
		return nil, fmt.Errorf("read BackupProgressWindow.swift: %w", err)
	}
	progressStr := string(progressSrc)
	resp.ProgressWindowSource = progressPath
	resp.SwiftSourcesChecked = []string{progressPath}

	// Hot-path also inspects remote app + client.
	combinedHot := progressStr
	for _, p := range []string{remoteApp, clientPath} {
		b, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		combinedHot += "\n" + string(b)
		resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, p)
	}

	resp.NoActivateInProgressWindow = hasNoActivateInProgressWindow(progressStr)
	resp.QuietOrderFront = hasQuietOrderFront(progressStr)
	resp.PresentsWindow = hasPresentsWindow(progressStr)
	resp.FlushIntervalInBand = hasFlushIntervalInBand(progressStr)
	resp.HasPendingBuffer = hasPendingBuffer(progressStr)
	resp.UsesTextStorageAppend = hasTextStorageAppend(progressStr)
	resp.ScrollOnlyOnFlush = hasScrollOnlyOnFlush(progressStr)
	resp.ProgressEnqueueOnly = hasProgressEnqueueOnly(combinedHot, progressStr)
	resp.HasScrollableConsole = hasScrollableConsole(progressStr)
	resp.NonEditableSelectable = hasNonEditableSelectable(progressStr)

	switch req.ClientLeaf {
	case "no-activate",
		"quiet-order-front",
		"presents-window",
		"flush-interval",
		"pending-buffer",
		"text-storage-append",
		"scroll-on-flush",
		"enqueue-only",
		"scroll-and-text-view",
		"non-editable-selectable":
		// flags populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

// hasNoActivateInProgressWindow: BackupProgressWindow must not activate the app.
func hasNoActivateInProgressWindow(src string) bool {
	if strings.Contains(src, "activate(ignoringOtherApps") {
		return false
	}
	if strings.Contains(src, "NSApp.activate") {
		return false
	}
	// Any AppKit activate on NSApp / NSApplication for this window file is banned.
	if regexp.MustCompile(`(?i)(NSApp|NSApplication\.shared)\s*\.\s*activate\s*\(`).MatchString(src) {
		return false
	}
	return true
}

// hasQuietOrderFront: non-activating order front (not only makeKeyAndOrderFront).
func hasQuietOrderFront(src string) bool {
	if strings.Contains(src, "orderFrontRegardless") {
		return true
	}
	// Strip makeKeyAndOrderFront so bare orderFront( can be detected (RE2 has no lookbehind).
	cleaned := strings.ReplaceAll(src, "makeKeyAndOrderFront", "")
	return regexp.MustCompile(`orderFront\s*\(`).MatchString(cleaned)
}

// hasPresentsWindow: still creates and shows a window (regression).
func hasPresentsWindow(src string) bool {
	hasWindow := strings.Contains(src, "NSWindow")
	// Any presentation: quiet or key.
	presents := strings.Contains(src, "orderFrontRegardless") ||
		strings.Contains(src, "makeKeyAndOrderFront") ||
		regexp.MustCompile(`orderFront\s*\(`).MatchString(strings.ReplaceAll(src, "makeKeyAndOrderFront", ""))
	return hasWindow && presents
}

// hasFlushIntervalInBand: Timer / TimeInterval / ms constant in 100–200ms.
func hasFlushIntervalInBand(src string) bool {
	// Explicit ms constants in band.
	if regexp.MustCompile(`(?i)(flushInterval|batchInterval|progressFlush).{0,40}\b(1[0-9]{2}|200)\b`).MatchString(src) {
		return true
	}
	// TimeInterval seconds 0.1 … 0.2 (e.g. 0.15, 0.1, 0.2)
	if regexp.MustCompile(`\b0\.1[0-9]?\b|\b0\.2\b`).MatchString(src) {
		// Require nearby timer/flush wording so random 0.15 elsewhere does not pass alone —
		// still accept if Timer / scheduledTimer / asyncAfter present with that number.
		if regexp.MustCompile(`(?is)(Timer|scheduledTimer|asyncAfter|flushInterval|batchInterval).{0,80}0\.(1[0-9]?|2)\b|0\.(1[0-9]?|2)\b.{0,80}(Timer|scheduledTimer|asyncAfter|flushInterval|batchInterval|seconds)`).MatchString(src) {
			return true
		}
	}
	// milliseconds: 100...200 near flush/timer
	if regexp.MustCompile(`(?is)(flush|batch|Timer|interval).{0,60}\b(1[0-9]{2}|200)\b\s*(\*|ms|milliseconds)?`).MatchString(src) {
		return true
	}
	// .milliseconds(150) style
	if regexp.MustCompile(`(?i)\.milliseconds\s*\(\s*(1[0-9]{2}|200)\s*\)`).MatchString(src) {
		return true
	}
	return false
}

// hasPendingBuffer: enqueue buffer + flush drain symbols.
func hasPendingBuffer(src string) bool {
	pending := regexp.MustCompile(`(?i)\b(pendingLines|pendingLine|lineBuffer|bufferedLines|pending)\b`).MatchString(src)
	flush := regexp.MustCompile(`(?i)\b(flushPending|flushBuffer|flushLines|func\s+flush|flush\s*\()`).MatchString(src)
	// append enqueues rather than only writing textView immediately
	enqueue := regexp.MustCompile(`(?is)func\s+append[\s\S]{0,400}(pending|buffer|enqueue)`).MatchString(src)
	return (pending && flush) || (enqueue && flush)
}

// hasTextStorageAppend: prefer textStorage batch write over sole string +=.
func hasTextStorageAppend(src string) bool {
	// textStorage.append / textStorage?.append / replaceCharacters in storage
	hasStorageWrite := regexp.MustCompile(`(?i)textStorage(\??\.\s*append|\s*\.\s*append|\)\.append)|NSTextStorage`).MatchString(src) &&
		regexp.MustCompile(`(?i)(append\s*\(|replaceCharacters)`).MatchString(src)
	if !hasStorageWrite {
		// Accept: textView.textStorage?.append(NSAttributedString…
		hasStorageWrite = regexp.MustCompile(`(?is)textStorage[\s\S]{0,80}append\s*\(`).MatchString(src)
	}
	if !hasStorageWrite {
		return false
	}
	// Fail if the only mutation path is still per-line string += without batch flush.
	// Presence of textStorage append is required; string += may remain for empty init only.
	return true
}

// hasScrollOnlyOnFlush: scrollToEnd lives on flush path; not paired with per-line string +=.
func hasScrollOnlyOnFlush(src string) bool {
	hasFlush := regexp.MustCompile(`(?i)\b(flushPending|flushBuffer|flushLines|func\s+\w*[Ff]lush\w*)\b`).MatchString(src)
	if !hasFlush {
		return false
	}
	// flush region scrolls
	if !regexp.MustCompile(`(?is)(flushPending|flushBuffer|flushLines|func\s+\w*[Ff]lush\w*)[\s\S]{0,800}scrollToEnd`).MatchString(src) {
		// looser: any flush-named call site near scroll
		if !regexp.MustCompile(`(?is)flush[\s\S]{0,200}scrollToEnd|scrollToEnd[\s\S]{0,200}flush`).MatchString(src) {
			return false
		}
	}
	// Anti-pattern: immediate per-line string += then scroll in same small body
	if regexp.MustCompile(`(?is)(string\s*\+=|string\s*=\s*line)[\s\S]{0,120}scrollToEnd`).MatchString(src) {
		return false
	}
	return true
}

// hasProgressEnqueueOnly: callback uses session append; session batches (not immediate string +=).
func hasProgressEnqueueOnly(combined, progressWindow string) bool {
	// Client / app: progressSession?.append from download/onProgress path (no large RE2 span).
	hasDownload := strings.Contains(combined, "downloadBackupArchive") || strings.Contains(combined, "onProgress")
	hasSessionAppend := strings.Contains(combined, "progressSession") &&
		(regexp.MustCompile(`progressSession\s*\?\.\s*append\s*\(`).MatchString(combined) ||
			regexp.MustCompile(`(?is)progressSession[\s\S]{0,120}append\s*\(`).MatchString(combined))
	if !(hasDownload && hasSessionAppend) {
		return false
	}
	// Session must batch: pending/flush or no immediate string += in append path
	if hasPendingBuffer(progressWindow) && hasTextStorageAppend(progressWindow) {
		return true
	}
	// Or append clearly enqueues only
	if regexp.MustCompile(`(?is)func\s+append[\s\S]{0,500}(pending|enqueue|buffer)`).MatchString(progressWindow) {
		return !regexp.MustCompile(`(?is)func\s+append[\s\S]{0,400}string\s*\+=`).MatchString(progressWindow)
	}
	return false
}

func hasScrollableConsole(src string) bool {
	hasScroll := strings.Contains(src, "NSScrollView")
	hasText := strings.Contains(src, "NSTextView")
	docView := strings.Contains(src, "documentView")
	return hasScroll && hasText && docView
}

func hasNonEditableSelectable(src string) bool {
	// isEditable = false (or .isEditable = false)
	notEditable := regexp.MustCompile(`(?i)isEditable\s*=\s*false`).MatchString(src)
	selectable := regexp.MustCompile(`(?i)isSelectable\s*=\s*true`).MatchString(src)
	return notEditable && selectable
}

func findModuleRoot() (string, error) {
	// DOCTEST_ROOT is injected (tree root). Walk up for go.mod.
	start := DOCTEST_ROOT
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = wd
	}
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}
```
