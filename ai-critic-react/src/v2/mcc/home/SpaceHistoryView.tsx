import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import { fetchAnalysisReports, type AnalysisReportSummary } from '../../../api/diskAnalysis';
import { formatBytes } from '../../../api/processes';
import SpaceLineChart from './SpaceLineChart';
import SpaceSparkline from './SpaceSparkline';
import { deltaTone, duration, fileCount, reclaimableTotal, shortWhen, signedBytes } from './spaceTone';
import './SystemView.css';
import './SpaceAnalysis.css';

const KEEP = 50;

export default function SpaceHistoryView() {
    const navigate = useNavigate();
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

    const ordered = reports.slice().reverse();
    const points = ordered.map(r => ({ label: shortWhen(r.scanned_at).slice(0, 5), value: reclaimableTotal(r) }));
    const newest = reports[0] ?? null;
    const oldest = reports[reports.length - 1] ?? null;
    const span = newest && oldest ? reclaimableTotal(newest) - reclaimableTotal(oldest) : 0;

    const open = (id: string) => {
        navigate({ pathname: '..', search: `?report=${encodeURIComponent(id)}` }, { relative: 'path' });
    };

    return (
        <div className="mcc-system">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button
                    className="mcc-back-btn"
                    onClick={() => navigate('..', { relative: 'path' })}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}
                >
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0 }}>History</h2>
                <span className="mcc-system-live paused">{reports.length}</span>
            </div>
            <div className="mcc-system-sub">
                newest {KEEP} kept · newest 5 keep directory detail · every row is a finished scan
            </div>

            {error && <div className="mcc-system-alert">{error}</div>}
            {loading && reports.length === 0 && <div className="mcc-system-empty">Reading reports…</div>}
            {!loading && reports.length === 0 && !error && (
                <div className="mcc-system-empty">No reports yet. Run a scan from the analysis page.</div>
            )}

            {reports.length > 1 && (
                <div className="mcc-space-card">
                    <div className="mcc-space-card-head">
                        <div className="mcc-space-card-title">Reclaimable space</div>
                        <span className={`mcc-space-delta ${deltaTone(span)}`}>{signedBytes(span)}</span>
                    </div>
                    <SpaceLineChart points={points} color="#22d3ee" highlight={points.length - 1} />
                </div>
            )}

            {reports.length > 0 && <div className="mcc-space-sect">REPORTS<span>newest first</span></div>}

            <div className="mcc-space-hist">
                {reports.map((report, index) => {
                    const older = reports[index + 1] ?? null;
                    const value = reclaimableTotal(report);
                    const delta = older ? value - reclaimableTotal(older) : null;
                    const window = reports.slice(index, index + 6).reverse();
                    const values = window.map(r => reclaimableTotal(r));
                    return (
                        <button
                            key={report.id}
                            className={`mcc-space-hist-row${index === 0 ? ' current' : ''}`}
                            onClick={() => open(report.id)}
                        >
                            <div className="mcc-space-hist-main">
                                <div className="mcc-space-hist-when">{shortWhen(report.scanned_at)}</div>
                                <div className="mcc-space-hist-meta">
                                    {duration(report.scan_ms)} · {fileCount(report.files)}
                                    {!report.complete ? ' · partial' : report.entries_dropped ? ' · detail trimmed' : ''}
                                </div>
                            </div>
                            {values.length > 1 && <SpaceSparkline values={values} />}
                            <div className="mcc-space-hist-right">
                                <div className="mcc-space-hist-bytes">{formatBytes(value)}</div>
                                <div className="mcc-space-hist-sub">
                                    {delta === null ? 'first' : signedBytes(delta)}
                                </div>
                            </div>
                        </button>
                    );
                })}
            </div>

            <div className="mcc-space-foot">
                tap a row to open that report · older reports keep only the category rollup, so their directory list is
                gone
            </div>
        </div>
    );
}
