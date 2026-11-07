import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { FolderIcon } from '../../../pure-view/icons/FolderIcon';
import { fetchAnalysisReports, fetchAnalysisStatus } from '../../../api/diskAnalysis';
import { formatBytes } from '../../../api/processes';
import SpaceSparkline from './SpaceSparkline';
import { reclaimableTotal, relativeTime } from './spaceTone';

const REFRESH_MS = 30000;

/** Entry point from the disk page into the space analysis + its history. */
export default function SpaceEntryCard() {
    const navigate = useNavigate();
    const [hydrated, setHydrated] = useState(false);
    const [running, setRunning] = useState(false);
    const [percent, setPercent] = useState(0);
    const [lastAt, setLastAt] = useState<string | undefined>(undefined);
    const [count, setCount] = useState(0);
    const [values, setValues] = useState<number[]>([]);
    const [cancelledAt, setCancelledAt] = useState<string | undefined>(undefined);
    const [error, setError] = useState('');

    useEffect(() => {
        let alive = true;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const status = await fetchAnalysisStatus();
                if (!alive) return;
                setRunning(status.running);
                setPercent(status.job?.percent ?? 0);
                setLastAt(status.last_completed_at);
                setCount(status.report_count);
                setCancelledAt(
                    status.last_attempt && status.last_attempt.outcome !== 'completed'
                        ? status.last_attempt.at
                        : undefined
                );
                setError('');
                if (status.report_count > 0) {
                    const list = await fetchAnalysisReports(8);
                    if (!alive) return;
                    setValues(list.reports.slice().reverse().map(reclaimableTotal));
                }
            } catch (err) {
                if (alive) setError(err instanceof Error ? err.message : String(err));
            } finally {
                if (alive) {
                    setHydrated(true);
                    timer = window.setTimeout(tick, REFRESH_MS);
                }
            }
        };
        void tick();
        return () => {
            alive = false;
            if (timer) window.clearTimeout(timer);
        };
    }, []);

    const reclaimable = values.length > 0 ? values[values.length - 1] : 0;
    let subtitle = 'not analysed yet';
    if (running) {
        subtitle = `scanning now · ${Math.round(percent)}%`;
    } else if (count > 0) {
        subtitle = `${relativeTime(lastAt)} · ${formatBytes(reclaimable)} reclaimable`;
    } else if (cancelledAt) {
        subtitle = `last try stopped ${relativeTime(cancelledAt)} · nothing stored`;
    } else if (!hydrated) {
        subtitle = 'reading scan state…';
    }

    return (
        <button
            type="button"
            className={`mcc-space-entry${running ? ' running' : ''}`}
            onClick={() => navigate('space')}
        >
            <span className="mcc-space-entry-ico">
                <FolderIcon />
            </span>
            <span className="mcc-space-entry-body">
                <span className="mcc-space-entry-title">
                    Space analysis
                    {count > 0 && <span className="mcc-space-entry-count">History {count}</span>}
                </span>
                <span className="mcc-space-entry-sub">{error ? `unavailable · ${error}` : subtitle}</span>
            </span>
            {!running && values.length > 1 && <SpaceSparkline values={values} width={56} height={22} />}
            <span className="mcc-space-entry-go">›</span>
        </button>
    );
}
