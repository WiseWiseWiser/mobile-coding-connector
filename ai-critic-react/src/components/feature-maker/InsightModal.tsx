import type { Insight } from './types';

interface InsightModalProps {
    insight: Insight;
    onClose: () => void;
}

export function InsightModal({ insight, onClose }: InsightModalProps) {
    return (
        <div className="fm-insight-modal-overlay" onClick={onClose}>
            <div className="fm-insight-modal" onClick={e => e.stopPropagation()}>
                <div className="fm-modal-header">
                    <span className={`fm-modal-type ${insight.type}`}>
                        {insight.type === 'decision' ? 'Decision' : insight.type === 'action' ? 'Action' : 'Question'}
                    </span>
                    <button className="fm-modal-close" onClick={onClose}>×</button>
                </div>
                <div className="fm-modal-body">
                    <h3>{insight.title}</h3>
                    <p>{insight.description}</p>
                    <div className="fm-modal-agent">
                        <span className="fm-modal-agent-label">From:</span>
                        <span className="fm-modal-agent-name">{insight.agent}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
