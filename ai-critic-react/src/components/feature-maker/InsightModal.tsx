import type { Insight } from './types';
import { ModalOverlay } from '../../pure-view/modals';
import './InsightModal.css';

interface InsightModalProps {
    insight: Insight;
    onClose: () => void;
}

export function InsightModal({ insight, onClose }: InsightModalProps) {
    return (
        <ModalOverlay onClose={onClose}>
            <div className="modal-header">
                <span className={`insight-modal-type ${insight.type}`}>
                    {insight.type === 'decision' ? 'Decision' : insight.type === 'action' ? 'Action' : 'Question'}
                </span>
                <button className="modal-close-btn" onClick={onClose}>×</button>
            </div>
            <div className="modal-body">
                <h3>{insight.title}</h3>
                <p>{insight.description}</p>
                <div className="insight-modal-agent">
                    <span className="insight-modal-agent-label">From:</span>
                    <span className="insight-modal-agent-name">{insight.agent}</span>
                </div>
            </div>
        </ModalOverlay>
    );
}
