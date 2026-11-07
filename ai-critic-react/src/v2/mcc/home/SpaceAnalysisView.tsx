import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import {
    cancelAnalysis,
    fetchAnalysisReport,
    fetchAnalysisReports,
    fetchAnalysisStatus,
    startAnalysis,
    type AnalysisReport,
    type AnalysisReportSummary,
    type AnalysisStatus,
} from '../../../api/diskAnalysis';
import { formatBytes } from '../../../api/processes';
import SpaceProgressCard from './SpaceProgressCard';
import SpaceResultCard from './SpaceResultCard';
import SpaceEntryList from './SpaceEntryList';
import { clockTime, isStale, reclaimableTotal, relativeTime, shortWhen } from './spaceTone';
import './SystemView.css';
import './SpaceAnalysis.css';

const RUNNING_MS = 1000;
const IDLE_MS = 10000;

function errText(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
}

export default function SpaceAnalysisView() {
    const navigate = useNavigate();
    const [params, setParams] = useSearchParams();
    const wantedId = params.get('report');

    const [status, setStatus] = useState<AnalysisStatus | null>(null);
    const [reports, setReports] = useState<AnalysisReportSummary[]>([]);
    const [report, setReport] = useState<AnalysisReport | null>(null);
    const [error, setError] = useState('');
    const [loaded, setLoaded] = useState(false);
    const [stopping, setStopping] = useState(false);
    const [starting, setStarting] = useState(false);
    const [nonce, setNonce] = useState(0);

    const running = status?.running ?? false;

    // Poll /status: the server owns the job, so a refresh resumes at the same
    // percent instead of restarting the scan.
    useEffect(() => {
        let alive = true;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchAnalysisStatus();
                if (!alive) return;
                setStatus(next);
                setError('');
            } catch (err) {
                if (alive) setError(errText(err));
            } finally {
                if (alive) setLoaded(true);
            }
            if (alive) timer = window.setTimeout(tick, running ? RUNNING_MS : IDLE_MS);
        };
        void tick();
        return () => {
            alive = false;
            if (timer) window.clearTimeout(timer);
        };
    }, [running, nonce]);

    // Refresh the list whenever a new report lands (or the first one shows up).
    const reportsKey = status ? `${status.report_count}:${status.last_completed_at ?? ''}` : '';

    useEffect(() => {
        if (!status) return;
        let alive = true;
        fetchAnalysisReports(50)
            .then(res => {
                if (alive) setReports(res.reports);
            })
            .catch(err => {
                if (alive) setError(errText(err));
            });
        return () => {
            alive = false;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [reportsKey]);

    const selectedId = wantedId ?? status?.latest?.id ?? '';
    useEffect(() => {
        if (!selectedId) {
            setReport(null);
            return;
        }
        if (report?.id === selectedId) return;
        let alive = true;
        fetchAnalysisReport(selectedId)
            .then(next => {
                if (alive) setReport(next);
            })
            .catch(err => {
                if (alive) setError(errText(err));
            });
        return () => {
            alive = false;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [selectedId]);

    useEffect(() => {
        if (running) setStopping(false);
    }, [running, status?.last_attempt?.at]);

    const start = async () => {
        setStarting(true);
        setError('');
        try {
            const result = await startAnalysis();
            setStatus(prev =>
                prev
                    ? { ...prev, running: true, job: result.job }
                    : { running: true, job: result.job, report_count: 0 }
            );
            setNonce(n => n + 1);
        } catch (err) {
            setError(errText(err));
        } finally {
            setStarting(false);
        }
    };

    const cancel = async () => {
        setStopping(true);
        setError('');
        try {
            const cancelled = await cancelAnalysis();
            if (!cancelled) setError('the scan had already finished');
            setNonce(n => n + 1);
        } catch (err) {
            setStopping(false);
            setError(errText(err));
        }
    };

    const latest = reports[0] ?? status?.latest ?? null;
    const showing = report && (!wantedId || report.id === wantedId) ? report : null;
    const isHistoric = Boolean(wantedId && showing);
    const showingIndex = showing ? reports.findIndex(r => r.id === showing.id) : -1;
    const previous = showingIndex >= 0 ? reports[showingIndex + 1] ?? null : reports[1] ?? null;
    const attemptNote = attemptLine(status);

    const openTrend = (categoryId: string) => {
        navigate(`trend/${encodeURIComponent(categoryId)}`);
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
                <h2 style={{ margin: 0 }}>Space analysis</h2>
                <span className={`mcc-system-live${running ? '' : ' paused'}`}>
                    {running ? `${RUNNING_MS / 1000}s` : loaded ? `${IDLE_MS / 1000}s` : '…'}
                </span>
                {running ? (
                    <button
                        type="button"
                        className="mcc-system-chip danger"
                        onClick={() => void cancel()}
                        disabled={stopping}
                    >
                        {stopping ? 'Stopping…' : 'Stop'}
                    </button>
                ) : (
                    <>
                        {reports.length > 0 && (
                            <button
                                type="button"
                                className="mcc-system-chip"
                                onClick={() => navigate('history')}
                            >
                                History {reports.length}
                            </button>
                        )}
                        <button
                            type="button"
                            className="mcc-system-chip on"
                            onClick={() => void start()}
                            disabled={starting}
                        >
                            {starting ? 'Starting…' : 'Scan'}
                        </button>
                    </>
                )}
            </div>

            <div className="mcc-system-sub">
                {status ? statusSubtitle(status) : 'reading server state…'}
                {attemptNote ? ` · ${attemptNote}` : ''}
            </div>

            {error && <div className="mcc-system-alert">{error}</div>}
            {!loaded && !status && <div className="mcc-system-empty">Loading scan state…</div>}

            {isHistoric && (
                <div className="mcc-space-note">
                    <span>
                        Viewing the report from {shortWhen(showing?.scanned_at)}.
                    </span>
                    <button type="button" className="mcc-system-chip" onClick={() => setParams({})}>
                        Back to latest
                    </button>
                </div>
            )}

            {running && status?.job && <SpaceProgressCard job={status.job} stopping={stopping} />}

            {!running && !showing && loaded && (
                <div className="mcc-system-empty">
                    {latest
                        ? 'Select a report from the history to inspect it.'
                        : 'No scan yet. Run one to see what is taking the space.'}
                </div>
            )}

            {!running && latest && !showing && reports.length > 0 && (
                <SpaceResultCard summary={latest} previous={previous} onPickCategory={openTrend} />
            )}

            {showing && (
                <>
                    {running && <div className="mcc-space-sect">PREVIOUS REPORT</div>}
                    <SpaceResultCard summary={showing} previous={previous} onPickCategory={openTrend} />
                    <div className="mcc-space-sect">
                        BIGGEST DIRECTORIES
                        <span>{showing.entries?.length ?? 0} recorded</span>
                    </div>
                    <SpaceEntryList report={showing} onPickCategory={openTrend} />
                </>
            )}

            {!running && latest && reports.length > 1 && (
                <div className="mcc-space-links">
                    <button type="button" className="mcc-space-link" onClick={() => navigate('history')}>
                        <span>History</span>
                        <span>{reports.length} reports · {formatBytes(reclaimableTotal(latest))} reclaimable now</span>
                    </button>
                </div>
            )}

            {status?.last_attempt?.outcome === 'cancelled' && !running && (
                <div className="mcc-space-foot">
                    the last try was stopped on purpose, so it wrote no report · {relativeTime(status.last_attempt.at)}
                </div>
            )}
            {isStale(status?.last_completed_at) && !running && reports.length > 0 && (
                <div className="mcc-space-stale">
                    the newest report is from {shortWhen(status?.last_completed_at)} · run a scan for a current picture
                </div>
            )}
            <div className="mcc-space-foot">
                reports live in ~/.ai-critic/disk/ · newest 5 keep directory detail · sizes are apparent bytes
            </div>
        </div>
    );
}

function statusSubtitle(status: AnalysisStatus): string {
    const parts: string[] = [];
    if (status.last_completed_at) {
        parts.push(`last completed ${relativeTime(status.last_completed_at)}`);
    } else if (!status.running) {
        parts.push('no completed scan yet');
    }
    if (status.report_count > 0) {
        parts.push(`${status.report_count} ${status.report_count === 1 ? 'report' : 'reports'} stored`);
    }
    if (status.running && status.job) {
        parts.push(`scanning since ${clockTime(status.job.started_at)}`);
    }
    return parts.join(' · ') || 'idle';
}

function attemptLine(status: AnalysisStatus | null): string {
    const attempt = status?.last_attempt;
    if (!attempt) return '';
    if (attempt.outcome === 'completed') return '';
    if (attempt.outcome === 'cancelled') return `last try stopped ${relativeTime(attempt.at)}`;
    if (attempt.outcome === 'interrupted') return `last try interrupted ${relativeTime(attempt.at)}`;
    if (attempt.outcome === 'failed') return `last try failed ${relativeTime(attempt.at)}`;
    return '';
}
