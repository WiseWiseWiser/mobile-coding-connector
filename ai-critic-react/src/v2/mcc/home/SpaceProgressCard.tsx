import { formatBytes } from '../../../api/processes';
import type { AnalysisJob } from '../../../api/diskAnalysis';
import { clockTime, fileCount } from './spaceTone';

interface Props {
    job: AnalysisJob;
    stopping: boolean;
}

/** Live card for the scan that is running on the server right now. */
export default function SpaceProgressCard({ job, stopping }: Props) {
    const percent = Math.max(0, Math.min(99, Math.round(job.percent || 0)));
    const phase = stopping ? 'stopping' : job.phase || 'scanning';
    return (
        <div className="mcc-space-card mcc-space-prog">
            <div className="mcc-space-prog-head">
                <div>
                    <div className="mcc-space-prog-title">
                        {stopping ? 'Stopping the scan…' : 'Scanning on the server'}
                    </div>
                    <div className="mcc-space-prog-id">job #{job.id}</div>
                </div>
                <div className="mcc-space-prog-pct">{percent}%</div>
            </div>
            <div className="mcc-space-prog-bar">
                <i style={{ width: `${Math.max(percent, 2)}%` }} />
            </div>
            <div className="mcc-space-prog-cells">
                <div>
                    <span>Phase</span>
                    <b>{phase}</b>
                </div>
                <div>
                    <span>Walked</span>
                    <b>{formatBytes(job.walked_bytes)}</b>
                </div>
                <div>
                    <span>Files</span>
                    <b>{fileCount(job.files)}</b>
                </div>
                <div>
                    <span>No access</span>
                    <b>{job.no_access_dirs}</b>
                </div>
            </div>
            <div className="mcc-space-prog-foot">
                started {clockTime(job.started_at) || 'just now'} · progress is written to disk, so a refresh
                picks it up at {percent}% instead of restarting
            </div>
        </div>
    );
}
