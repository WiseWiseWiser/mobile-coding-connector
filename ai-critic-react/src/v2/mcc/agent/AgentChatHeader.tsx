import { useState, useRef, useEffect } from 'react';
import { BackIcon } from '../../icons';

export interface ModelOption {
    id: string;
    name: string;
    is_default?: boolean;
    is_current?: boolean;
}

export interface AgentChatHeaderProps {
    agentName: string;
    projectName: string | null;
    onStop?: () => void;
    onBack?: () => void;
    stopLabel?: string;
    modelName?: string;
    contextPercent?: number;
    availableModels?: ModelOption[];
    onModelChange?: (modelId: string) => void;
}

export function AgentChatHeader({ agentName, projectName: _projectName, onStop, onBack, stopLabel = 'Stop', modelName, contextPercent, availableModels, onModelChange }: AgentChatHeaderProps) {
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    // Close dropdown on outside click
    useEffect(() => {
        if (!dropdownOpen) return;
        const handler = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                setDropdownOpen(false);
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [dropdownOpen]);

    const hasModels = availableModels && availableModels.length > 0 && onModelChange;

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
                    <div className="mcc-agent-model-info" ref={dropdownRef}>
                        <span
                            className={`mcc-agent-model-name${hasModels ? ' mcc-agent-model-clickable' : ''}`}
                            onClick={hasModels ? () => setDropdownOpen(!dropdownOpen) : undefined}
                        >
                            {modelName}
                            {hasModels && <span className="mcc-agent-model-chevron">▾</span>}
                        </span>
                        {contextPercent !== undefined && (
                            <span className="mcc-agent-context-usage">{contextPercent}%</span>
                        )}
                        {dropdownOpen && hasModels && (
                            <div className="mcc-agent-model-dropdown">
                                {availableModels.map(m => (
                                    <button
                                        key={m.id}
                                        className={`mcc-agent-model-option${m.id === modelName ? ' mcc-agent-model-option-active' : ''}`}
                                        onClick={() => {
                                            onModelChange(m.id);
                                            setDropdownOpen(false);
                                        }}
                                    >
                                        <span className="mcc-agent-model-option-name">{m.name}</span>
                                        <span className="mcc-agent-model-option-id">{m.id}</span>
                                        {m.is_default && <span className="mcc-agent-model-badge">default</span>}
                                        {m.is_current && <span className="mcc-agent-model-badge mcc-agent-model-badge-current">current</span>}
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>
                )}
            </div>
            {onStop && <button className="mcc-agent-stop-btn" onClick={onStop}>{stopLabel}</button>}
        </div>
    );
}
