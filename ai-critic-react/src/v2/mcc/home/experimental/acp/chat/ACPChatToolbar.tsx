import { ModelSelector, type ModelOption } from '../../../../../../pure-view/selector/ModelSelector';
import type { ConnectionStatus } from './ACPChatTypes';

export interface ACPChatToolbarProps {
    models: ModelOption[];
    selectedModel?: { modelID: string; providerID: string };
    onModelSelect: (m: { modelID: string; providerID: string }) => void;
    modelPlaceholder: string;
    yoloMode: boolean;
    onYoloToggle: () => void;
    debugMode: boolean;
    onDebugToggle: () => void;
    dir: string;
    status: ConnectionStatus;
}

export function ACPChatToolbar({
    models,
    selectedModel,
    onModelSelect,
    modelPlaceholder,
    yoloMode,
    onYoloToggle,
    debugMode,
    onDebugToggle,
    dir,
    status,
}: ACPChatToolbarProps) {
    return (
        <div className="acp-ui-toolbar">
            <ModelSelector
                models={models}
                currentModel={selectedModel}
                onSelect={onModelSelect}
                placeholder={modelPlaceholder}
                disabled={models.length === 0}
            />
            <button
                className={`acp-ui-debug-toggle ${yoloMode ? 'active' : ''}`}
                onClick={onYoloToggle}
                title="Toggle YOLO mode (bypass all confirmations)"
            >
                YOLO
            </button>
            <button
                className={`acp-ui-debug-toggle ${debugMode ? 'active' : ''}`}
                onClick={onDebugToggle}
                title="Toggle debug mode"
            >
                Debug
            </button>
            <div className="acp-ui-cwd-container">
                <span className="acp-ui-cwd-label">Project Dir:</span>
                <span className="acp-ui-cwd-display">
                    {dir || (status === 'connected' || status === 'connecting' ? 'loading...' : 'N/A')}
                </span>
            </div>
        </div>
    );
}
