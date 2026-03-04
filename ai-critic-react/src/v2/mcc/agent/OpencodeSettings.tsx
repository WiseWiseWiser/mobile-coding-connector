import { useState, useEffect } from 'react';
import {
    fetchOpencodeAuthStatus,
    fetchOpencodeConfig,
    fetchAgentEffectivePath,
    fetchOpencodeProviders,
    fetchOpencodeSettings,
    fetchOpencodeWebStatus,
    fetchOpencodeAuthKeys,
    setOpencodeAuthKey,
    deleteOpencodeAuthKey,
    updateAgentConfig,
    updateOpencodeSettings,
    startOpencodeWebServerStreaming,
    stopOpencodeWebServerStreaming,
    unmapOpencodeDomain,
    mapOpencodeDomainStreaming,
} from '../../../api/agents';
import type { OpencodeAuthStatus, AgentEffectivePath, AgentSessionInfo, OpencodeSettings, OpencodeWebStatus, OpencodeAuthKeyEntry } from '../../../api/agents';
import { fetchProviders } from '../../../api/ports';
import { AgentChatHeader } from './AgentChatHeader';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { useReconnectingStreamingAction } from '../../../hooks/useReconnectingStreamingAction';
import { ModelSelector, type ModelOption } from '../components/ModelSelector';
import { ProviderKeysSection } from './opencode/settings/ProviderKeysSection';
import { WebServerSection } from './opencode/settings/WebServerSection';
import { DomainConfigSection } from './opencode/settings/DomainConfigSection';

export interface OpencodeSettingsProps {
    agentId: string;
    session: AgentSessionInfo | null;
    projectName: string | null;
    onBack: () => void;
    onRefreshAgents?: () => void;
}

export function OpencodeSettings({ agentId, session, projectName, onBack, onRefreshAgents }: OpencodeSettingsProps) {
    const [authStatus, setAuthStatus] = useState<OpencodeAuthStatus | null>(null);
    const [webStatus, setWebStatus] = useState<OpencodeWebStatus | null>(null);
    const [effectivePath, setEffectivePath] = useState<AgentEffectivePath | null>(null);

    const [savedSettings, setSavedSettings] = useState<OpencodeSettings>({});
    const [defaultDomain, setDefaultDomain] = useState('');
    const [binaryPath, setBinaryPath] = useState('');
    const [password, setPassword] = useState('');
    const [webServerPort, setWebServerPort] = useState(4096);
    const [webServerEnabled, setWebServerEnabled] = useState(false);
    const [authProxyEnabled, setAuthProxyEnabled] = useState(false);

    const [savedModel, setSavedModel] = useState<{ modelID: string; providerID: string } | null>(null);
    const [selectedModel, setSelectedModel] = useState<{ modelID: string; providerID: string } | null>(null);
    const [models, setModels] = useState<ModelOption[]>([]);
    const [defaultModel, setDefaultModel] = useState('');

    const [authKeys, setAuthKeys] = useState<OpencodeAuthKeyEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [success, setSuccess] = useState('');
    const [error, setError] = useState('');

    const [webServerActionState, webServerActionControls] = useStreamingAction((result) => {
        if (!result.ok) setError(result.message);
        else setSuccess(result.message);
        setTimeout(refreshWebStatus, 1000);
    });

    const [domainMapped, setDomainMapped] = useState(false);
    const [mappedUrl, setMappedUrl] = useState('');
    const [availableProviders, setAvailableProviders] = useState<Array<{ id: string; name: string; available: boolean }>>([]);

    const [domainMappingState, domainMappingControls] = useReconnectingStreamingAction((result) => {
        if (result.ok) {
            setDomainMapped(true);
            if (result.publicUrl) setMappedUrl(result.publicUrl);
            setSuccess(result.message);
        } else {
            setError(result.message);
        }
        setTimeout(refreshWebStatus, 1000);
    }, { maxReconnects: 20, reconnectDelayMs: 2000 });

    const hasChanges = selectedModel?.modelID !== savedModel?.modelID || selectedModel?.providerID !== savedModel?.providerID;
    const hasSettingsChanges = webServerEnabled !== (savedSettings.web_server?.enabled ?? false)
        || defaultDomain !== (savedSettings.default_domain || '')
        || binaryPath !== (savedSettings.binary_path || '')
        || webServerPort !== (savedSettings.web_server?.port || 4096)
        || password !== (savedSettings.web_server?.password || '')
        || authProxyEnabled !== (savedSettings.web_server?.auth_proxy_enabled ?? false);

    useEffect(() => { loadAllData(); }, [session?.id]);

    const loadAllData = async () => {
        setLoading(true);
        try {
            const [settings, webStat, auth, pathInfo, keys] = await Promise.all([
                fetchOpencodeSettings(),
                fetchOpencodeWebStatus(),
                fetchOpencodeAuthStatus(),
                fetchAgentEffectivePath(agentId),
                fetchOpencodeAuthKeys().catch(() => [] as OpencodeAuthKeyEntry[]),
            ]);

            setSavedSettings(settings);
            setDefaultDomain(settings.default_domain || '');
            setBinaryPath(settings.binary_path || '');
            setPassword(settings.web_server?.password || '');
            setWebServerPort(settings.web_server?.port || 4096);
            setWebServerEnabled(settings.web_server?.enabled ?? false);
            setAuthProxyEnabled(settings.web_server?.auth_proxy_enabled ?? false);
            setWebStatus(webStat);
            setAuthStatus(auth);
            setEffectivePath(pathInfo);
            setAuthKeys(keys);

            let savedModelKey: { modelID: string; providerID: string } | null = null;
            if (settings.model) {
                const parts = settings.model.split('/');
                savedModelKey = parts.length >= 2
                    ? { providerID: parts[0], modelID: parts[1] }
                    : { providerID: '', modelID: settings.model };
            }
            setSavedModel(savedModelKey);
            setSelectedModel(savedModelKey);

            try {
                const providers = await fetchProviders();
                setAvailableProviders(providers.filter(p => p.available && (p.id === 'cloudflare_owned' || p.id === 'cloudflare_tunnel')));
            } catch { /* ignore */ }

            if (session) {
                const [config, providers] = await Promise.all([
                    fetchOpencodeConfig(session.id),
                    fetchOpencodeProviders(session.id),
                ]);
                const currentModelFromServer = config.model?.modelID
                    ? { modelID: config.model.modelID, providerID: config.model.providerID || '' }
                    : null;
                const modelToUse = savedModelKey || currentModelFromServer;
                setSavedModel(modelToUse);
                setSelectedModel(modelToUse);

                const allModels: ModelOption[] = [];
                let defModel = '';
                for (const provider of providers.providers) {
                    for (const [id, model] of Object.entries(provider.models)) {
                        allModels.push({
                            id, name: model.name || id,
                            providerId: provider.id,
                            providerName: provider.name || provider.id,
                            is_default: providers.default?.[provider.id] === id,
                        });
                    }
                    if (providers.default?.[provider.id]) defModel = providers.default[provider.id];
                }
                allModels.sort((a, b) => a.providerName.localeCompare(b.providerName) || a.name.localeCompare(b.name));
                setModels(allModels);
                setDefaultModel(defModel);
            }
        } catch (err) {
            console.error('Failed to load settings:', err);
        } finally {
            setLoading(false);
        }
    };

    const refreshWebStatus = async () => {
        try { setWebStatus(await fetchOpencodeWebStatus()); } catch { /* ignore */ }
    };

    const refreshAuthKeys = async () => {
        try { setAuthKeys(await fetchOpencodeAuthKeys()); } catch { /* ignore */ }
    };

    const handleSaveKey = async (provider: string, key: string) => {
        await setOpencodeAuthKey(provider, key);
        setSuccess(`Key for ${provider} saved`);
        await refreshAuthKeys();
    };

    const handleDeleteKey = async (provider: string) => {
        await deleteOpencodeAuthKey(provider);
        setSuccess(`Key for ${provider} deleted`);
        await refreshAuthKeys();
    };

    const handleSaveSessionModel = async () => {
        if (!session || !selectedModel) return;
        setSaving(true); setError(''); setSuccess('');
        try {
            await updateAgentConfig(session.id, { model: { modelID: selectedModel.modelID } });
            setSavedModel(selectedModel);
            setSavedSettings({ ...savedSettings, model: `${selectedModel.providerID}/${selectedModel.modelID}` });
            setSuccess('Model updated successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update model');
        } finally { setSaving(false); }
    };

    const handleSaveSettings = async () => {
        setSaving(true); setError(''); setSuccess('');
        try {
            const currentConfig = savedSettings.web_server || { enabled: false, port: 4096 };
            const nextSettings = {
                ...savedSettings,
                default_domain: defaultDomain,
                binary_path: binaryPath,
                web_server: { ...currentConfig, enabled: webServerEnabled, port: webServerPort, password, auth_proxy_enabled: authProxyEnabled },
            };
            await updateOpencodeSettings(nextSettings);
            setSavedSettings(nextSettings);
            setEffectivePath(await fetchAgentEffectivePath(agentId));
            onRefreshAgents?.();
            setSuccess('Settings saved successfully');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save settings');
        } finally { setSaving(false); }
    };

    const handleCancel = () => {
        setSelectedModel(savedModel ? { ...savedModel } : null);
        setDefaultDomain(savedSettings.default_domain || '');
        setBinaryPath(savedSettings.binary_path || '');
        setPassword(savedSettings.web_server?.password || '');
        setWebServerEnabled(savedSettings.web_server?.enabled ?? false);
        setAuthProxyEnabled(savedSettings.web_server?.auth_proxy_enabled ?? false);
        setError(''); setSuccess('');
    };

    const handleWebServerControl = async (action: 'start' | 'stop') => {
        setError(''); setSuccess('');
        const streamFn = action === 'start' ? startOpencodeWebServerStreaming : stopOpencodeWebServerStreaming;
        await webServerActionControls.run(() => streamFn());
    };

    const handleMapDomain = async () => {
        setError(''); setSuccess('');
        await domainMappingControls.run((sessionId, logIndex) => mapOpencodeDomainStreaming(undefined, sessionId, logIndex));
    };

    const handleUnmapDomain = async () => {
        setError(''); setSuccess('');
        try {
            const resp = await unmapOpencodeDomain();
            if (resp.success) { setDomainMapped(false); setMappedUrl(''); setSuccess(resp.message); }
            else setError(resp.message);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to unmap domain');
        }
    };

    return (
        <div className="mcc-agent-view">
            {session ? (
                <AgentChatHeader agentName={session.agent_name} projectName={projectName} onBack={onBack} />
            ) : (
                <div className="mcc-section-header">
                    <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                    <h2>OpenCode Settings</h2>
                </div>
            )}
            <div className="mcc-agent-header" style={{ paddingTop: 4 }}>
                <h2>Settings</h2>
            </div>

            {loading ? (
                <div className="mcc-agent-loading">Loading settings...</div>
            ) : (
                <div className="mcc-agent-settings-form">
                    {/* Binary Path */}
                    <div className="mcc-agent-settings-field" style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
                        <label className="mcc-agent-settings-label">OpenCode Binary Path</label>
                        <div className="mcc-agent-settings-hint" style={{ marginBottom: 8, fontSize: '13px', color: '#94a3b8' }}>
                            Custom path to the OpenCode binary. Leave empty to use PATH resolution.
                        </div>
                        <input type="text" value={binaryPath} onChange={(e) => setBinaryPath(e.target.value)}
                            placeholder={`e.g. /usr/local/bin/${agentId}`} disabled={saving}
                            style={{ width: '100%', padding: '10px 12px', background: '#1e293b',
                                border: binaryPath !== (savedSettings.binary_path || '') ? '1px solid #3b82f6' : '1px solid #334155',
                                borderRadius: 8, color: '#e2e8f0', fontSize: '14px' }} />
                        <div style={{ marginTop: 10, padding: '10px 12px',
                            background: effectivePath?.found ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                            border: `1px solid ${effectivePath?.found ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                            borderRadius: 8, fontFamily: 'monospace', fontSize: '13px',
                            color: effectivePath?.found ? '#86efac' : '#fca5a5', wordBreak: 'break-all' }}>
                            {effectivePath?.found ? (
                                <>
                                    <div>Effective Path: {effectivePath.effective_path}</div>
                                    {effectivePath.version && <div style={{ marginTop: 4, color: '#94a3b8' }}>Version: {effectivePath.version}</div>}
                                </>
                            ) : `Effective Path: Not found${effectivePath?.error ? ` (${effectivePath.error})` : ''}`}
                        </div>
                    </div>

                    {/* Login Status */}
                    <div className="mcc-agent-settings-field" style={{ marginBottom: 20 }}>
                        <label className="mcc-agent-settings-label">Login Status</label>
                        <div style={{ padding: '12px 14px',
                            background: authStatus?.authenticated ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                            border: `1px solid ${authStatus?.authenticated ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                            borderRadius: 8 }}>
                            <div style={{ color: authStatus?.authenticated ? '#86efac' : '#fca5a5', fontWeight: 600,
                                marginBottom: authStatus?.providers?.length ? 8 : 0 }}>
                                {authStatus?.authenticated ? '✓ Authenticated' : '✗ Not authenticated'}
                            </div>
                            {authStatus?.providers && authStatus.providers.length > 0 && (
                                <div style={{ fontSize: '13px', color: '#94a3b8' }}>
                                    <strong>Providers:</strong>
                                    <ul style={{ margin: '8px 0 0 0', paddingLeft: 20 }}>
                                        {authStatus.providers.map(p => (
                                            <li key={p.name} style={{ marginBottom: 4 }}>
                                                {p.name}
                                                {p.has_api_key && <span style={{ color: '#86efac', marginLeft: 8 }}>(configured)</span>}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}
                            {!authStatus?.authenticated && (
                                <div style={{ fontSize: '13px', color: '#94a3b8', marginTop: 8 }}>
                                    Run <code style={{ background: '#1e293b', padding: '2px 6px', borderRadius: 4 }}>opencode auth login</code> to authenticate.
                                </div>
                            )}
                        </div>
                    </div>

                    <ProviderKeysSection authKeys={authKeys} onSaveKey={handleSaveKey} onDeleteKey={handleDeleteKey} />

                    <WebServerSection
                        webStatus={webStatus}
                        webServerEnabled={webServerEnabled}
                        webServerPort={webServerPort}
                        password={password}
                        authProxyEnabled={authProxyEnabled}
                        saving={saving}
                        savedWebServer={savedSettings.web_server}
                        actionState={webServerActionState}
                        onRefresh={refreshWebStatus}
                        onControl={handleWebServerControl}
                        onEnabledChange={setWebServerEnabled}
                        onPortChange={setWebServerPort}
                        onPasswordChange={setPassword}
                        onAuthProxyChange={setAuthProxyEnabled}
                    />

                    <DomainConfigSection
                        defaultDomain={defaultDomain}
                        savedDefaultDomain={savedSettings.default_domain || ''}
                        saving={saving}
                        hasSettingsChanges={hasSettingsChanges}
                        webStatus={webStatus}
                        availableProviders={availableProviders}
                        domainMapped={domainMapped}
                        mappedUrl={mappedUrl}
                        domainMappingState={domainMappingState}
                        onDomainChange={setDefaultDomain}
                        onMapDomain={handleMapDomain}
                        onUnmapDomain={handleUnmapDomain}
                        onSaveSettings={handleSaveSettings}
                        onCancel={handleCancel}
                    />

                    {/* Preferred Model */}
                    <div className="mcc-agent-settings-field">
                        <label className="mcc-agent-settings-label">Preferred Model</label>
                        <div className="mcc-agent-settings-hint" style={{ marginBottom: 8 }}>
                            Select the AI model to use for this session.
                        </div>
                        <div style={{ border: hasChanges ? '1px solid #3b82f6' : '1px solid #334155', borderRadius: 8, padding: '2px' }}>
                            <ModelSelector models={models} currentModel={selectedModel || undefined}
                                onSelect={(model) => setSelectedModel(model)}
                                placeholder={models.length === 0 ? "Start a session to see available models" : "Select a model..."}
                                disabled={saving || models.length === 0} />
                        </div>
                        {savedModel && models.length > 0 && (
                            <div style={{ marginTop: 8, fontSize: '13px', color: '#94a3b8' }}>
                                Saved: <strong style={{ color: '#e2e8f0' }}>
                                    {models.find(m => m.id === savedModel.modelID && m.providerId === savedModel.providerID)?.name || savedModel.modelID}
                                </strong>
                                {savedModel.modelID === defaultModel && <span style={{ color: '#64748b', marginLeft: 4 }}>(default)</span>}
                            </div>
                        )}
                    </div>

                    {hasChanges && session && (
                        <div style={{ marginTop: 16, display: 'flex', gap: 12, padding: '12px',
                            background: 'rgba(59, 130, 246, 0.1)', borderRadius: 8, border: '1px solid rgba(59, 130, 246, 0.3)' }}>
                            <button onClick={handleSaveSessionModel} disabled={saving}
                                style={{ flex: 1, padding: '10px 16px', background: '#3b82f6', opacity: saving ? 0.7 : 1,
                                    border: 'none', borderRadius: 6, color: '#fff', fontSize: '14px', fontWeight: 500,
                                    cursor: saving ? 'not-allowed' : 'pointer' }}>
                                {saving ? 'Saving...' : 'Save'}
                            </button>
                            <button onClick={handleCancel} disabled={saving}
                                style={{ flex: 1, padding: '10px 16px', background: 'transparent', border: '1px solid #475569',
                                    borderRadius: 6, color: '#94a3b8', fontSize: '14px', fontWeight: 500,
                                    cursor: saving ? 'not-allowed' : 'pointer' }}>
                                Cancel
                            </button>
                        </div>
                    )}

                    {error && (
                        <div style={{ marginTop: 12, padding: '10px 14px', background: 'rgba(239, 68, 68, 0.1)',
                            border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 8, color: '#fca5a5', fontSize: '13px' }}>
                            {error}
                        </div>
                    )}
                    {success && (
                        <div style={{ marginTop: 12, padding: '10px 14px', background: 'rgba(34, 197, 94, 0.1)',
                            border: '1px solid rgba(34, 197, 94, 0.3)', borderRadius: 8, color: '#86efac', fontSize: '13px' }}>
                            {success}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
