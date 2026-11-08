import type { CleanupRunRow } from './spaceCleanupTone';

interface Props {
    rows: CleanupRunRow[];
}

function statusText(status: CleanupRunRow['status']): string {
    if (status === 'running') return 'running';
    if (status === 'skipped') return 'skipped';
    if (status === 'warn') return 'partial';
    return 'done';
}

/** Live per-class progress while a run streams. */
export default function SpaceCleanupRunList({ rows }: Props) {
    if (rows.length === 0) {
        return <div className="mcc-cl-pending">waiting for the first class…</div>;
    }
    return (
        <div className="mcc-run">
            {rows.map(row => (
                <div key={row.id} className={`mcc-run-row ${row.status}`}>
                    <span className={`mcc-run-dot ${row.status}`} />
                    <div className="mcc-run-main">
                        <div className="mcc-run-name">{row.name}</div>
                        {row.detail && <div className="mcc-run-note">{row.detail}</div>}
                    </div>
                    <span className={`mcc-run-tag ${row.status}`}>{statusText(row.status)}</span>
                </div>
            ))}
        </div>
    );
}
