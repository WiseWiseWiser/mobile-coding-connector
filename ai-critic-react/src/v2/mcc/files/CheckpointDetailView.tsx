import { useState, useEffect } from 'react';
import {
    fetchCheckpointDetail,
    fetchCheckpointDiff,
} from '../../../api/checkpoints';
import type { CheckpointDetail, FileDiff } from '../../../api/checkpoints';
import { DiffViewer } from '../../DiffViewer';
import { statusBadge } from './utils';
import './FilesView.css';

export interface CheckpointDetailViewProps {
    projectName: string;
    checkpointId: number;
    onBack: () => void;
}

export function CheckpointDetailView({ projectName, checkpointId, onBack }: CheckpointDetailViewProps) {
    const [detail, setDetail] = useState<CheckpointDetail | null>(null);
    const [diffs, setDiffs] = useState<FileDiff[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchCheckpointDetail(projectName, checkpointId),
            fetchCheckpointDiff(projectName, checkpointId),
        ])
            .then(([detailData, diffData]) => {
                setDetail(detailData);
                setDiffs(diffData || []);
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [projectName, checkpointId]);

    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2>{detail?.name || `Checkpoint #${checkpointId}`}</h2>
            </div>
            {loading ? (
                <div className="mcc-files-empty">Loading...</div>
            ) : !detail ? (
                <div className="mcc-files-empty">Checkpoint not found.</div>
            ) : (
                <>
                    <div className="mcc-checkpoint-detail-meta">
                        <span>{new Date(detail.timestamp).toLocaleString()}</span>
                        <span>{detail.files.length} file{detail.files.length !== 1 ? 's' : ''}</span>
                    </div>
                    <div className="mcc-changed-files-list">
                        {detail.files.map(f => (
                            <div key={f.path} className="mcc-changed-file-item mcc-changed-file-item-readonly">
                                {statusBadge(f.status)}
                                <span className="mcc-changed-file-path">{f.path}</span>
                            </div>
                        ))}
                    </div>

                    {/* File Diffs */}
                    <div className="mcc-checkpoint-section-label">File Diffs</div>
                    <DiffViewer diffs={diffs} />
                </>
            )}
        </div>
    );
}
