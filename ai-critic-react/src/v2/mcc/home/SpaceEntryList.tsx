import { formatBytes } from '../../../api/processes';
import type { AnalysisReport } from '../../../api/diskAnalysis';
import { categoryColor, percentOf } from './spaceTone';

interface Props {
    report: AnalysisReport;
    onPickCategory: (categoryId: string) => void;
}

const MAX_ROWS = 20;

/** Biggest directories of one report; keeps the scan honest about where bytes sit. */
export default function SpaceEntryList({ report, onPickCategory }: Props) {
    const entries = (report.entries ?? []).slice().sort((a, b) => b.bytes - a.bytes);
    if (entries.length === 0) {
        return (
            <div className="mcc-space-card mcc-space-empty">
                {report.entries_dropped
                    ? 'Directory detail was trimmed for this report; only the category rollup is kept.'
                    : 'No directory crossed the 256 MB threshold in this scan.'}
            </div>
        );
    }

    const rows = entries.slice(0, MAX_ROWS);
    const largest = rows[0]?.bytes || 1;
    const labels = new Map(report.categories.map(c => [c.id, c.label]));

    return (
        <div className="mcc-space-card mcc-space-entries">
            {rows.map(entry => (
                <button
                    key={`${entry.category}:${entry.path}`}
                    className="mcc-space-entry-row"
                    onClick={() => onPickCategory(entry.category)}
                >
                    <div className="mcc-space-entry-top">
                        <span
                            className="mcc-space-entry-cat"
                            style={{ color: categoryColor(entry.category) }}
                        >
                            {labels.get(entry.category) ?? entry.category}
                        </span>
                        <span className="mcc-space-entry-bytes">{formatBytes(entry.bytes)}</span>
                    </div>
                    <div className="mcc-space-entry-path">{entry.path}</div>
                    <div className="mcc-space-cat-bar">
                        <i
                            style={{
                                width: `${percentOf(entry.bytes, largest)}%`,
                                background: categoryColor(entry.category),
                            }}
                        />
                    </div>
                    {entry.reclaimable && <span className="mcc-space-tag cache">freeable</span>}
                </button>
            ))}
            {entries.length > rows.length && (
                <div className="mcc-space-foot">showing the {rows.length} largest of {entries.length} directories</div>
            )}
        </div>
    );
}
