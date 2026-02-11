import { ModelSelector, type ModelOption } from '../components/ModelSelector';

export interface AgentChatHeaderProps {
    agentName: string;
    projectName: string | null;
    onStop?: () => void;
    onBack?: () => void;
    stopLabel?: string;
    modelName?: string;
    contextPercent?: number;
    availableModels?: ModelOption[];
    currentModel?: { modelID: string; providerID: string };
    onModelChange?: (model: { modelID: string; providerID: string }) => void;
}

export function AgentChatHeader({ 
    agentName, 
    projectName: _projectName, 
    onStop, 
    onBack, 
    stopLabel = 'Stop', 
    contextPercent,
    availableModels,
    currentModel,
    onModelChange 
}: AgentChatHeaderProps) {
    const hasModels = availableModels && availableModels.length > 0 && onModelChange;

    return (
        <div className="mcc-agent-chat-header">
            {onBack && (
                <button className="mcc-agent-back-btn" onClick={onBack} title="Back to agents">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M19 12H5M12 19l-7-7 7-7"/>
                    </svg>
                </button>
            )}
            <div className="mcc-agent-chat-title">
                <span>{agentName}</span>
                {hasModels && currentModel && (
                    <div className="mcc-agent-model-info">
                        <ModelSelector
                            models={availableModels}
                            currentModel={currentModel}
                            onSelect={onModelChange!}
                            placeholder="Select model..."
                        />
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

export type { ModelOption } from '../components/ModelSelector';
