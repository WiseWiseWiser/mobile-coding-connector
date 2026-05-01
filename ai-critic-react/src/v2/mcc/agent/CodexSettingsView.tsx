import { useEffect, useState } from 'react';
import { AgentChatHeader } from './AgentChatHeader';
import {
    CODEX_DEFAULT_APPROVAL_POLICY,
    CODEX_DEFAULT_SANDBOX,
    fetchCodexModels,
    loadCodexApprovalPolicy,
    loadCodexDefaultModel,
    loadCodexSandbox,
    saveCodexApprovalPolicy,
    saveCodexDefaultModel,
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
    const [sandbox, setSandbox] = useState(loadCodexSandbox);
    const [approvalPolicy, setApprovalPolicy] = useState(loadCodexApprovalPolicy);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        fetchCodexModels()
            .then(data => {
                setModels(data.models);
                setModel(current => current || data.currentModel || data.models[0]?.id || '');
            })
            .catch(err => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setLoading(false));
    }, []);

    const handleModelChange = (value: string) => {
        setModel(value);
        saveCodexDefaultModel(value);
    };

    const handleSandboxChange = (value: string) => {
        setSandbox(value);
        saveCodexSandbox(value);
    };

    const handleApprovalPolicyChange = (value: string) => {
        setApprovalPolicy(value);
        saveCodexApprovalPolicy(value);
    };

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
