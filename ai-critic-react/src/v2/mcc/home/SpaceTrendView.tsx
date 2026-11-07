import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import { fetchAnalysisReports, type AnalysisReportSummary } from '../../../api/diskAnalysis';
import { formatBytes } from '../../../api/processes';
import SpaceLineChart, { type ChartPoint } from './SpaceLineChart';
import {
    categoryColor,
    categoryBytes,
    deltaTone,
    percentOf,
    reclaimableTotal,
    shortWhen,
    signedBytes,
} from './spaceTone';
import './SystemView.css';
import './SpaceAnalysis.css';

const KEEP = 50;

interface Group {
    schema: number;
    points: ChartPoint[];
    values: { id: string; label: string; scannedAt: string; value: number | null }[];
}

export default function SpaceTrendView() {
    const navigate = useNavigate();
    const { category = '' } = useParams();
    const [reports, setReports] = useState<AnalysisReportSummary[]>([]);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        let alive = true;
        fetchAnalysisReports(KEEP)
            .then(res => {
                if (alive) setReports(res.reports);
            })
            .catch(err => {
                if (alive) setError(err instanceof Error ? err.message : String(err));
            })
            .finally(() => {
                if (alive) setLoading(false);
            });
        return () => {
            alive = false;
        };
    }, []);

    const newest = reports[0] ?? null;
    const found = newest?.categories.find(c => c.id === category) ?? null;
    const label = found?.label ?? category;
    const color = categoryColor(category);

    const ordered = reports.slice().reverse();
    const groups = splitBySchema(ordered, category);
    const measured = groups.flatMap(g => g.values).filter(v => v.value !== null) as {
        id: string;
        label: string;
        scannedAt: string;
        value: number;
    }[];
    const current = measured.length > 0 ? measured[measured.length - 1] : null;
    const peak = measured.reduce<(typeof measured)[number] | null>(
        (best, item) => (best === null || item.value > best.value ? item : best),
        null
    );
    const usedTotal = newest ? newest.used_bytes : 0;

    return (
        <div className="mcc-system">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button
                    className="mcc-back-btn"
                    onClick={() => navigate(-1)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}
                >
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0, color }}>{label}</h2>
                <span className="mcc-system-live paused">{measured.length}</span>
            </div>
            <div className="mcc-system-sub">
                {current
                    ? `now ${formatBytes(current.value)}${usedTotal > 0 ? ` · ${percentOf(current.value, usedTotal)}% of used` : ''}`
                    : 'no measurements yet'}
                {peak && peak.id !== current?.id ? ` · peak ${formatBytes(peak.value)} on ${shortWhen(peak.scannedAt)}` : ''}
                {found && found.reclaimable > 0 ? ` · ${formatBytes(found.reclaimable)} freeable` : ''}
            </div>

            {error && <div className="mcc-system-alert">{error}</div>}
            {loading && reports.length === 0 && <div className="mcc-system-empty">Reading reports…</div>}
            {!loading && measured.length === 0 && !error && (
                <div className="mcc-system-empty">
                    No report measured {label} yet. Run more scans; the curve fills in as history grows.
                </div>
            )}

            {groups.map((group, index) => {
                const first = group.values[0]?.scannedAt ?? '';
                const last = group.values[group.values.length - 1]?.scannedAt ?? '';
                const showRange = group.values.length > 1 && first !== last;
                return (
                    <div className="mcc-space-card" key={`${group.schema}-${index}`}>
                        <div className="mcc-space-card-head">
                            <div>
                                <div className="mcc-space-card-title">
                                    {groups.length > 1 ? `Measurement v${group.schema}` : 'Bytes over time'}
                                </div>
                                {showRange && (
                                    <div className="mcc-space-meta">
                                        {shortWhen(first)} → {shortWhen(last)}
                                    </div>
                                )}
                            </div>
                            <span className="mcc-space-tag">apparent bytes</span>
                        </div>
                        <SpaceLineChart
                            points={group.points}
                            color={color}
                            highlight={index === groups.length - 1 ? group.points.length - 1 : -1}
                        />
                    </div>
                );
            })}

            {groups.length > 1 && (
                <div className="mcc-space-note">
                    The scan rules changed between these runs, so the older points live on their own axis instead of
                    pretending the jump is real growth.
                </div>
            )}

            {measured.length > 0 && <div className="mcc-space-sect">EVERY REPORT<span>newest first</span></div>}

            <div className="mcc-space-hist">
                {ordered
                    .slice()
                    .reverse()
                    .map((report, index, all) => {
                        const value = categoryBytes(report, category);
                        const olderSummary = all[index + 1] ?? null;
                        const olderValue = olderSummary ? categoryBytes(olderSummary, category) : null;
                        const delta = value !== null && olderValue !== null ? value - olderValue : null;
                        return (
                            <div key={report.id} className="mcc-space-hist-row static">
                                <div className="mcc-space-hist-main">
                                    <div className="mcc-space-hist-when">{shortWhen(report.scanned_at)}</div>
                                    <div className="mcc-space-hist-meta">
                                        report {report.id}
                                        {!report.complete ? ' · partial' : ''}
                                    </div>
                                </div>
                                <div className="mcc-space-hist-right">
                                    <div className="mcc-space-hist-bytes">
                                        {value === null ? 'not measured' : formatBytes(value)}
                                    </div>
                                    <div className={`mcc-space-hist-sub ${delta === null ? '' : deltaTone(delta)}`}>
                                        {delta === null ? '—' : signedBytes(delta)}
                                    </div>
                                </div>
                            </div>
                        );
                    })}
            </div>

            <div className="mcc-space-foot">
                {formatBytes(reclaimableTotal(newest))} of the newest report is reclaimable across all categories ·
                a missing point stays a gap, never a zero
            </div>
        </div>
    );
}

/** Consecutive runs of the same schema, so incomparable numbers never share an axis. */
function splitBySchema(reports: AnalysisReportSummary[], category: string): Group[] {
    const groups: Group[] = [];
    reports.forEach(report => {
        const value = report.categories.find(c => c.id === category);
        const bucket = groups[groups.length - 1];
        const item = {
            id: report.id,
            label: shortWhen(report.scanned_at).slice(0, 5),
            scannedAt: report.scanned_at,
            value: value ? value.bytes : null,
        };
        if (bucket && bucket.schema === report.schema) {
            bucket.values.push(item);
            bucket.points.push({ label: item.label, value: item.value });
            return;
        }
        groups.push({
            schema: report.schema,
            points: [{ label: item.label, value: item.value }],
            values: [item],
        });
    });
    return groups;
}
