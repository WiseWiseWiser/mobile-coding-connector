import { BackIcon } from '../../icons';

export interface AgentChatHeaderProps {
    agentName: string;
    projectName: string | null;
    onStop?: () => void;
    onBack?: () => void;
    stopLabel?: string;
    modelName?: string;
    contextPercent?: number;
}

export function AgentChatHeader({ agentName, projectName: _projectName, onStop, onBack, stopLabel = 'Stop', modelName, contextPercent }: AgentChatHeaderProps) {
    return (
        <div className="mcc-agent-chat-header">
            {onBack && (
                <button className="mcc-agent-back-btn" onClick={onBack} title="Back to agents">
                    <BackIcon />
                </button>
            )}
            <div className="mcc-agent-chat-title">
                <span>{agentName}</span>
                {modelName && (
                    <div className="mcc-agent-model-info">
                        <span className="mcc-agent-model-name">{modelName}</span>
                        {contextPercent !== undefined && (
                            <span className="mcc-agent-context-usage">{contextPercent}%</span>
                        )}
                    </div>
                )}
            </div>
            {onStop && <button className="mcc-agent-stop-btn" onClick={onStop}>{stopLabel}</button>}
        </div>
    );
}
