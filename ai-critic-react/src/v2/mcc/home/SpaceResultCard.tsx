import { formatBytes } from '../../../api/processes';
import type { AnalysisReport, AnalysisReportSummary } from '../../../api/diskAnalysis';
import {
    categoryColor,
    deltaTone,
    duration,
    fileCount,
    percentOf,
    reclaimableTotal,
    relativeTime,
    signedBytes,
    sortedCategories,
} from './spaceTone';

interface Props {
    summary: AnalysisReport | AnalysisReportSummary;
    previous?: AnalysisReportSummary | null;
    onPickCategory: (categoryId: string) => void;
}

/** One finished report: rollup bar, reclaimable total, category rows. */
export default function SpaceResultCard({ summary, previous, onPickCategory }: Props) {
    const categories = sortedCategories(summary);
    const total = categories.reduce((sum, c) => sum + c.bytes, 0) || 1;
    const largest = categories.length > 0 ? categories[0].bytes : 1;
    const reclaimable = reclaimableTotal(summary);
    const previousReclaimable = previous ? reclaimableTotal(previous) : null;
    const reclaimDelta = previousReclaimable === null ? null : reclaimable - previousReclaimable;
    const previousBytes = (id: string) => {
        if (!previous) return null;
        const found = previous.categories.find(c => c.id === id);
        return found ? found.bytes : null;
    };

    return (
        <div className="mcc-space-card">
            <div className="mcc-space-card-head">
                <div>
                    <div className="mcc-space-card-title">Report {summary.id}</div>
                    <div className="mcc-space-meta">
                        {relativeTime(summary.scanned_at)} · {duration(summary.scan_ms)} · {fileCount(summary.files)}
                        {summary.no_access_dirs > 0 ? ` · ${summary.no_access_dirs} dirs skipped` : ''}
                    </div>
                </div>
                <div className="mcc-space-badges">
                    {!summary.complete && <span className="mcc-space-badge warn">partial</span>}
                    {summary.truncated && <span className="mcc-space-badge warn">time budget</span>}
                    {summary.entries_dropped && <span className="mcc-space-badge">detail trimmed</span>}
                </div>
            </div>

            <div className="mcc-space-stack">
                {categories.map(c => (
                    <i
                        key={c.id}
                        style={{ width: `${percentOf(c.bytes, total)}%`, background: categoryColor(c.id) }}
                    />
                ))}
            </div>
            <div className="mcc-space-legend">
                {categories.map(c => (
                    <span key={c.id} className="mcc-space-lg">
                        <i style={{ background: categoryColor(c.id) }} />
                        {c.label}
                        <b>{formatBytes(c.bytes)}</b>
                    </span>
                ))}
            </div>

            <div className="mcc-space-reclaim">
                <div>
                    <div className="mcc-space-reclaim-value">≈ {formatBytes(reclaimable)} reclaimable</div>
                    <div className="mcc-space-reclaim-sub">
                        {reclaimDelta === null ? 'first report, no comparison yet' : `${signedBytes(reclaimDelta)} since the last scan`}
                        {' · '}
                        {formatBytes(summary.scanned_bytes)} walked of {formatBytes(summary.used_bytes)} used
                    </div>
                </div>
                {previous && (
                    <span className={`mcc-space-delta ${deltaTone(reclaimDelta ?? 0)}`}>
                        {signedBytes(reclaimDelta ?? 0)}
                    </span>
                )}
            </div>

            <div className="mcc-space-cats">
                {categories.map(c => {
                    const before = previousBytes(c.id);
                    const delta = before === null ? null : c.bytes - before;
                    return (
                        <button key={c.id} className="mcc-space-cat" onClick={() => onPickCategory(c.id)}>
                            <div className="mcc-space-cat-top">
                                <span className="mcc-space-cat-name">{c.label}</span>
                                {c.reclaimable > 0 ? (
                                    <span className="mcc-space-tag cache">{formatBytes(c.reclaimable)} freeable</span>
                                ) : (
                                    <span className="mcc-space-tag">keep</span>
                                )}
                                {delta !== null && delta !== 0 && (
                                    <span className={`mcc-space-delta ${deltaTone(delta)}`}>{signedBytes(delta)}</span>
                                )}
                                <span className="mcc-space-cat-bytes">{formatBytes(c.bytes)}</span>
                            </div>
                            <div className="mcc-space-cat-path">
                                {c.paths && c.paths.length > 0 ? c.paths.join(' · ') : 'no matching rule'}
                            </div>
                            <div className="mcc-space-cat-bar">
                                <i
                                    style={{
                                        width: `${percentOf(c.bytes, largest)}%`,
                                        background: categoryColor(c.id),
                                    }}
                                />
                            </div>
                        </button>
                    );
                })}
            </div>

            <div className="mcc-space-foot">
                apparent file sizes · APFS clones and purgeable space are not counted · tap a category for its
                trend
            </div>
        </div>
    );
}
