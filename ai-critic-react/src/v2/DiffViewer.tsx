import { useState } from 'react';
import type { FileDiff, DiffHunk, DiffLine } from '../api/checkpoints';
import './DiffViewer.css';

const DEFAULT_VISIBLE_LINES = 5;

interface DiffViewerProps {
    diffs: FileDiff[];
}

/**
 * Renders a list of file diffs in GitHub-style unified format.
 * Optimized for mobile screens.
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

// --- File Diff Card ---

function FileDiffCard({ diff }: { diff: FileDiff }) {
    const [expanded, setExpanded] = useState(false);

    const statusLabel = diff.status === 'added' ? 'A' : diff.status === 'deleted' ? 'D' : 'M';
    const statusCls = `diff-file-status diff-file-status-${diff.status}`;

    const addedCount = diff.hunks.reduce((sum, h) => sum + h.lines.filter(l => l.type === 'add').length, 0);
    const removedCount = diff.hunks.reduce((sum, h) => sum + h.lines.filter(l => l.type === 'delete').length, 0);

    // Collect all lines across hunks for truncation
    const allLines: { hunkIdx: number; line: DiffLine }[] = [];
    diff.hunks.forEach((hunk, hunkIdx) => {
        hunk.lines.forEach(line => {
            allLines.push({ hunkIdx, line });
        });
    });

    const totalLines = allLines.length;
    const needsTruncation = totalLines > DEFAULT_VISIBLE_LINES;
    const hiddenCount = needsTruncation ? totalLines - DEFAULT_VISIBLE_LINES : 0;

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
                <div className="diff-file-hunks">
                    {diff.hunks.map((hunk, i) => (
                        <HunkView
                            key={i}
                            hunk={hunk}
                            expanded={expanded}
                            maxLines={DEFAULT_VISIBLE_LINES}
                            hunkIdx={i}
                            allLines={allLines}
                        />
                    ))}
                    {needsTruncation && (
                        <button
                            className="diff-expand-btn"
                            onClick={() => setExpanded(!expanded)}
                        >
                            {expanded ? 'Collapse' : `Show ${hiddenCount} more line${hiddenCount !== 1 ? 's' : ''}`}
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
    expanded: boolean;
    maxLines: number;
    hunkIdx: number;
    allLines: { hunkIdx: number; line: DiffLine }[];
}

function HunkView({ hunk, expanded, maxLines, hunkIdx, allLines }: HunkViewProps) {
    const header = `@@ -${hunk.old_start},${hunk.old_lines} +${hunk.new_start},${hunk.new_lines} @@`;

    // Calculate which lines to show based on global line index
    const linesToShow = expanded ? hunk.lines : hunk.lines.filter((_, lineIdx) => {
        // Find global index of this line
        let globalIdx = 0;
        for (let i = 0; i < allLines.length; i++) {
            if (allLines[i].hunkIdx === hunkIdx) {
                const hunkLineIdx = i - allLines.findIndex(l => l.hunkIdx === hunkIdx);
                if (hunkLineIdx === lineIdx) {
                    globalIdx = i;
                    break;
                }
            }
        }
        return globalIdx < maxLines;
    });

    if (linesToShow.length === 0) {
        return null;
    }

    return (
        <div className="diff-hunk">
            <div className="diff-hunk-header">{header}</div>
            <div className="diff-hunk-lines">
                {linesToShow.map((line, i) => (
                    <LineView key={i} line={line} />
                ))}
            </div>
        </div>
    );
}

// --- Line View ---

function LineView({ line }: { line: DiffLine }) {
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
