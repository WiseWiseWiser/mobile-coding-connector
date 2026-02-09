import { useState, useMemo } from 'react';
import type { FileDiff, DiffHunk, DiffLine } from '../api/checkpoints';
import './DiffViewer.css';

const DEFAULT_VISIBLE_LINES = 5;
const LINES_PER_STEP = 10;

interface DiffViewerProps {
    diffs: FileDiff[];
}

/**
 * Renders a list of file diffs in GitHub-style unified format.
 * Optimized for mobile screens with iterative line-by-line navigation.
 */
export function DiffViewer({ diffs }: DiffViewerProps) {
    if (diffs.length === 0) {
        return <div className="diff-empty">No diffs to display</div>;
    }

    return (
        <div className="diff-viewer">
            {diffs.map(fileDiff => (
                <FileDiffCard key={fileDiff.path} diff={fileDiff} />
            ))}
        </div>
    );
}

// --- Types ---

interface IndexedLine extends DiffLine {
    globalIndex: number;
    hunkIndex: number;
    lineInHunk: number;
}

interface ViewportState {
    startLine: number;
    endLine: number;
    totalLines: number;
}

// --- File Diff Card ---

function FileDiffCard({ diff }: { diff: FileDiff }) {
    const statusLabel = diff.status === 'added' ? 'A' : diff.status === 'deleted' ? 'D' : 'M';
    const statusCls = `diff-file-status diff-file-status-${diff.status}`;

    const addedCount = diff.hunks.reduce((sum, h) => sum + h.lines.filter(l => l.type === 'add').length, 0);
    const removedCount = diff.hunks.reduce((sum, h) => sum + h.lines.filter(l => l.type === 'delete').length, 0);

    // Flatten all lines with global indices
    const allLines: IndexedLine[] = useMemo(() => {
        let globalIdx = 0;
        return diff.hunks.flatMap((hunk, hunkIdx) =>
            hunk.lines.map((line, lineIdx) => ({
                ...line,
                globalIndex: globalIdx++,
                hunkIndex: hunkIdx,
                lineInHunk: lineIdx
            }))
        );
    }, [diff]);

    const totalLines = allLines.length;

    // Initialize viewport showing first N lines
    const [viewport, setViewport] = useState<ViewportState>({
        startLine: 0,
        endLine: Math.min(DEFAULT_VISIBLE_LINES, totalLines),
        totalLines
    });

    // Calculate hidden lines
    const hiddenAbove = viewport.startLine;
    const hiddenBelow = viewport.totalLines - viewport.endLine;
    const hasHiddenAbove = hiddenAbove > 0;
    const hasHiddenBelow = hiddenBelow > 0;
    const isExpanded = viewport.endLine - viewport.startLine > DEFAULT_VISIBLE_LINES;

    // Viewport management functions
    const expandUp = () => {
        setViewport(prev => ({
            ...prev,
            startLine: Math.max(0, prev.startLine - LINES_PER_STEP)
        }));
    };

    const expandDown = () => {
        setViewport(prev => ({
            ...prev,
            endLine: Math.min(prev.totalLines, prev.endLine + LINES_PER_STEP)
        }));
    };

    const collapse = () => {
        // Reset to default view, keeping roughly centered on current view
        const currentCenter = Math.floor((viewport.startLine + viewport.endLine) / 2);
        const halfWindow = Math.floor(DEFAULT_VISIBLE_LINES / 2);
        const newStart = Math.max(0, currentCenter - halfWindow);
        const newEnd = Math.min(totalLines, newStart + DEFAULT_VISIBLE_LINES);

        setViewport({
            startLine: newStart,
            endLine: newEnd,
            totalLines
        });
    };

    // Get visible lines for rendering
    const visibleLines = useMemo(() => {
        return allLines.slice(viewport.startLine, viewport.endLine);
    }, [allLines, viewport]);

    // Group visible lines by hunk for rendering
    const visibleHunks = useMemo(() => {
        const hunkMap = new Map<number, IndexedLine[]>();
        visibleLines.forEach(line => {
            if (!hunkMap.has(line.hunkIndex)) {
                hunkMap.set(line.hunkIndex, []);
            }
            hunkMap.get(line.hunkIndex)!.push(line);
        });
        return hunkMap;
    }, [visibleLines]);

    return (
        <div className="diff-file-card">
            <div className="diff-file-header">
                <span className={statusCls}>{statusLabel}</span>
                <span className="diff-file-path">{diff.path}</span>
                <span className="diff-file-stats">
                    {addedCount > 0 && <span className="diff-stat-add">+{addedCount}</span>}
                    {removedCount > 0 && <span className="diff-stat-del">-{removedCount}</span>}
                </span>
            </div>
            {diff.hunks.length === 0 ? (
                <div className="diff-file-empty">Binary file or no content changes</div>
            ) : (
                <div className="diff-file-content">
                    {/* Load More Above button */}
                    {hasHiddenAbove && (
                        <button className="diff-load-btn diff-load-above" onClick={expandUp}>
                            <span className="diff-load-icon">↑</span>
                            <span className="diff-load-text">
                                {hiddenAbove <= LINES_PER_STEP
                                    ? `Load ${hiddenAbove} more line${hiddenAbove !== 1 ? 's' : ''}`
                                    : `Load ${LINES_PER_STEP} more lines`}
                            </span>
                            <span className="diff-load-count">({hiddenAbove} above)</span>
                        </button>
                    )}

                    {/* Diff content with viewport */}
                    <div className={`diff-viewport ${hasHiddenAbove ? 'diff-viewport-top-truncated' : ''} ${hasHiddenBelow ? 'diff-viewport-bottom-truncated' : ''}`}>
                        {diff.hunks.map((hunk, hunkIdx) => {
                            const hunkLines = visibleHunks.get(hunkIdx);
                            if (!hunkLines || hunkLines.length === 0) {
                                return null;
                            }
                            return (
                                <HunkView
                                    key={hunkIdx}
                                    hunk={hunk}
                                    hunkIdx={hunkIdx}
                                    lines={hunkLines}
                                />
                            );
                        })}
                    </div>

                    {/* Load More Below button */}
                    {hasHiddenBelow && (
                        <button className="diff-load-btn diff-load-below" onClick={expandDown}>
                            <span className="diff-load-icon">↓</span>
                            <span className="diff-load-text">
                                {hiddenBelow <= LINES_PER_STEP
                                    ? `Load ${hiddenBelow} more line${hiddenBelow !== 1 ? 's' : ''}`
                                    : `Load ${LINES_PER_STEP} more lines`}
                            </span>
                            <span className="diff-load-count">({hiddenBelow} below)</span>
                        </button>
                    )}

                    {/* Collapse button (only when expanded) */}
                    {isExpanded && (
                        <button className="diff-collapse-btn" onClick={collapse}>
                            <span className="diff-collapse-icon">−</span>
                            Show less
                        </button>
                    )}
                </div>
            )}
        </div>
    );
}

// --- Hunk View ---

interface HunkViewProps {
    hunk: DiffHunk;
    hunkIdx: number;
    lines: IndexedLine[];
}

function HunkView({ hunk, lines }: HunkViewProps) {
    const header = `@@ -${hunk.old_start},${hunk.old_lines} +${hunk.new_start},${hunk.new_lines} @@`;

    return (
        <div className="diff-hunk">
            <div className="diff-hunk-header">{header}</div>
            <div className="diff-hunk-lines">
                {lines.map((line) => (
                    <LineView key={line.globalIndex} line={line} />
                ))}
            </div>
        </div>
    );
}

// --- Line View ---

function LineView({ line }: { line: IndexedLine }) {
    const prefix = line.type === 'add' ? '+' : line.type === 'delete' ? '-' : ' ';
    const cls = `diff-line diff-line-${line.type}`;

    return (
        <div className={cls}>
            <span className="diff-line-num diff-line-num-old">{line.old_num || ''}</span>
            <span className="diff-line-num diff-line-num-new">{line.new_num || ''}</span>
            <span className="diff-line-prefix">{prefix}</span>
            <span className="diff-line-content">{line.content}</span>
        </div>
    );
}
