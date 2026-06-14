import { useEffect, useState } from 'react';
import { AgentChatHeader } from './AgentChatHeader';
import {
    CODEX_DEFAULT_APPROVAL_POLICY,
    CODEX_DEFAULT_SANDBOX,
    fetchCodexModels,
    loadCodexApprovalPolicy,
    loadCodexDefaultModel,
    loadCodexDefaultReasoningEffort,
    loadCodexSandbox,
    saveCodexApprovalPolicy,
    saveCodexDefaultModel,
    saveCodexDefaultReasoningEffort,
    saveCodexSandbox,
    type CodexModel,
} from './codexSettings';
import './AgentView.css';

export interface CodexSettingsViewProps {
    projectName: string | null;
    onBack: () => void;
}

export function CodexSettingsView({ projectName, onBack }: CodexSettingsViewProps) {
    const [models, setModels] = useState<CodexModel[]>([]);
    const [model, setModel] = useState(loadCodexDefaultModel);
    const [reasoningEffort, setReasoningEffort] = useState(loadCodexDefaultReasoningEffort);
    const [sandbox, setSandbox] = useState(loadCodexSandbox);
    const [approvalPolicy, setApprovalPolicy] = useState(loadCodexApprovalPolicy);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        fetchCodexModels()
            .then(data => {
                setModels(data.models);
                setModel(current => {
                    const nextModel = current || data.currentModel || data.models[0]?.id || '';
                    const modelInfo = findCodexModel(data.models, nextModel);
                    setReasoningEffort(currentEffort => currentEffort || data.currentReasoningLevel || modelInfo?.defaultReasoningLevel || modelInfo?.reasoningLevels?.[0] || '');
                    return nextModel;
                });
            })
            .catch(err => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setLoading(false));
    }, []);

    const handleModelChange = (value: string) => {
        setModel(value);
        saveCodexDefaultModel(value);
        const nextEffort = defaultReasoningEffortForModel(models, value);
        setReasoningEffort(nextEffort);
        saveCodexDefaultReasoningEffort(nextEffort);
    };

    const handleReasoningEffortChange = (value: string) => {
        setReasoningEffort(value);
        saveCodexDefaultReasoningEffort(value);
    };

    const handleSandboxChange = (value: string) => {
        setSandbox(value);
        saveCodexSandbox(value);
    };

    const handleApprovalPolicyChange = (value: string) => {
        setApprovalPolicy(value);
        saveCodexApprovalPolicy(value);
    };

    const selectedModel = findCodexModel(models, model);
    const reasoningLevels = selectedModel?.reasoningLevels || [];

    return (
        <div className="mcc-agent-view mcc-codex-settings-view">
            <AgentChatHeader agentName="Codex Settings" projectName={projectName} onBack={onBack} />
            <div className="mcc-codex-settings-content">
                {loading && <div className="mcc-agent-loading">Loading Codex settings...</div>}
                {error && <div className="mcc-agent-error">{error}</div>}
                {!loading && (
                    <>
                        <label className="mcc-codex-setting-row">
                            <span>
                                <strong>Default model</strong>
                                <small>Used when a chat starts or no session model is selected.</small>
                            </span>
                            <select value={model} onChange={event => handleModelChange(event.target.value)}>
                                <option value="">Codex default</option>
                                {models.map(item => (
                                    <option key={item.id} value={item.id}>{item.name}</option>
                                ))}
                            </select>
                        </label>
                        <label className="mcc-codex-setting-row">
                            <span>
                                <strong>Reasoning effort</strong>
                                <small>Controls how much reasoning Codex uses with the selected model.</small>
                            </span>
                            <select
                                value={reasoningEffort}
                                onChange={event => handleReasoningEffortChange(event.target.value)}
                                disabled={reasoningLevels.length === 0}
                            >
                                {reasoningLevels.length === 0 && <option value="">Model default</option>}
                                {reasoningLevels.map(level => (
                                    <option key={level} value={level}>{formatReasoningEffort(level)}</option>
                                ))}
                            </select>
                        </label>
                        <label className="mcc-codex-setting-row">
                            <span>
                                <strong>Permission</strong>
                                <small>Controls filesystem and command access for new Codex runs.</small>
                            </span>
                            <select value={sandbox} onChange={event => handleSandboxChange(event.target.value)}>
                                <option value="danger-full-access">Full access</option>
                                <option value="workspace-write">Workspace write</option>
                                <option value="read-only">Read only</option>
                            </select>
                        </label>
                        <label className="mcc-codex-setting-row">
                            <span>
                                <strong>Approval policy</strong>
                                <small>Controls whether Codex asks before running commands.</small>
                            </span>
                            <select value={approvalPolicy} onChange={event => handleApprovalPolicyChange(event.target.value)}>
                                <option value="never">Never ask</option>
                                <option value="on-request">Ask on request</option>
                                <option value="untrusted">Ask for untrusted commands</option>
                            </select>
                        </label>
                        <button
                            className="mcc-btn-secondary"
                            onClick={() => {
                                handleModelChange('');
                                handleReasoningEffortChange('');
                                handleSandboxChange(CODEX_DEFAULT_SANDBOX);
                                handleApprovalPolicyChange(CODEX_DEFAULT_APPROVAL_POLICY);
                            }}
                        >
                            Reset Defaults
                        </button>
                    </>
                )}
            </div>
        </div>
    );
}

function findCodexModel(models: CodexModel[], modelID: string): CodexModel | undefined {
    return models.find(item => item.id === modelID);
}

function defaultReasoningEffortForModel(models: CodexModel[], modelID: string): string {
    const model = findCodexModel(models, modelID);
    return model?.defaultReasoningLevel || model?.reasoningLevels?.[0] || '';
}

function formatReasoningEffort(value: string): string {
    if (!value) return 'Model default';
    if (value === 'xhigh') return 'Extra high';
    return `${value[0]?.toUpperCase() || ''}${value.slice(1)}`;
}
