import { useState, useEffect } from 'react';
import { fetchAIConfig, saveAIConfig, type AIProvider, type AIModel } from '../../../../api/ai';
import { EditIcon } from '../../../icons';
import { FlexInput } from '../../../../pure-view/FlexInput';
import './AIModelsSection.css';

export function AIModelsSection() {
    const [providers, setProviders] = useState<AIProvider[]>([]);
    const [models, setModels] = useState<AIModel[]>([]);
    const [defaultProvider, setDefaultProvider] = useState('');
    const [defaultModel, setDefaultModel] = useState('');
    const [usingNewFile, setUsingNewFile] = useState(false);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const [editingProviders, setEditingProviders] = useState(false);
    const [editingModels, setEditingModels] = useState(false);
    const [editingDefaults, setEditingDefaults] = useState(false);

    const [draftProviders, setDraftProviders] = useState<AIProvider[]>([]);
    const [draftModels, setDraftModels] = useState<AIModel[]>([]);
    const [draftDefaultProvider, setDraftDefaultProvider] = useState('');
    const [draftDefaultModel, setDraftDefaultModel] = useState('');

    const [newProviderName, setNewProviderName] = useState('');
    const [newProviderBaseURL, setNewProviderBaseURL] = useState('');
    const [newProviderAPIKey, setNewProviderAPIKey] = useState('');

    const [newModelProvider, setNewModelProvider] = useState('');
    const [newModelName, setNewModelName] = useState('');
    const [newModelDisplayName, setNewModelDisplayName] = useState('');

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = () => {
        setLoading(true);
        setError(null);
        fetchAIConfig()
            .then(cfg => {
                setProviders(cfg.providers || []);
                setModels(cfg.models || []);
                setDefaultProvider(cfg.default_provider || '');
                setDefaultModel(cfg.default_model || '');
                setDraftProviders(cfg.providers || []);
                setDraftModels(cfg.models || []);
                setDraftDefaultProvider(cfg.default_provider || '');
                setDraftDefaultModel(cfg.default_model || '');
                setUsingNewFile(cfg.using_new_file);
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
    };

    const persistConfig = async (newProviders: AIProvider[], newModels: AIModel[], newDefaultProvider: string, newDefaultModel: string, section: string) => {
        setSaving(section);
        setError(null);
        setSuccess(null);
        try {
            await saveAIConfig({
                providers: newProviders,
                models: newModels,
                default_provider: newDefaultProvider,
                default_model: newDefaultModel,
            });
            setProviders(newProviders);
            setModels(newModels);
            setDefaultProvider(newDefaultProvider);
            setDefaultModel(newDefaultModel);
            setUsingNewFile(true);
            setSuccess(`${section} saved successfully!`);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setSaving(null);
    };

    const startEditProviders = () => {
        setDraftProviders([...providers]);
        setNewProviderName('');
        setNewProviderBaseURL('');
        setNewProviderAPIKey('');
        setEditingProviders(true);
        setError(null);
        setSuccess(null);
    };

    const cancelEditProviders = () => {
        setEditingProviders(false);
        setDraftProviders([]);
        setError(null);
    };

    const saveEditProviders = async () => {
        const removedProviders = providers.filter(p => !draftProviders.some(dp => dp.name === p.name));
        let updatedModels = [...models];
        let updatedDefaultProvider = defaultProvider;
        let updatedDefaultModel = defaultModel;

        removedProviders.forEach(rp => {
            updatedModels = updatedModels.filter(m => m.provider !== rp.name);
            if (defaultProvider === rp.name) {
                updatedDefaultProvider = '';
                updatedDefaultModel = '';
            }
        });

        await persistConfig([...draftProviders], updatedModels, updatedDefaultProvider, updatedDefaultModel, 'Providers');
        setEditingProviders(false);
    };

    const handleAddDraftProvider = () => {
        const name = newProviderName.trim();
        const baseURL = newProviderBaseURL.trim();
        const apiKey = newProviderAPIKey.trim();

        if (!name || !baseURL) {
            setError('Provider name and base URL are required');
            return;
        }

        if (draftProviders.some(p => p.name === name)) {
            setError(`Provider "${name}" already exists`);
            return;
        }

        setDraftProviders([...draftProviders, { name, base_url: baseURL, api_key: apiKey }]);
        setNewProviderName('');
        setNewProviderBaseURL('');
        setNewProviderAPIKey('');
        setError(null);
    };

    const handleRemoveDraftProvider = (index: number) => {
        setDraftProviders(draftProviders.filter((_, i) => i !== index));
    };

    const startEditModels = () => {
        setDraftModels([...models]);
        setNewModelProvider('');
        setNewModelName('');
        setNewModelDisplayName('');
        setEditingModels(true);
        setError(null);
        setSuccess(null);
    };

    const cancelEditModels = () => {
        setEditingModels(false);
        setDraftModels([]);
        setError(null);
    };

    const saveEditModels = async () => {
        const removedModels = models.filter(m => !draftModels.some(dm => dm.provider === m.provider && dm.model === m.model));
        let updatedDefaultModel = defaultModel;

        removedModels.forEach(rm => {
            if (defaultModel === rm.model && defaultProvider === rm.provider) {
                updatedDefaultModel = '';
            }
        });

        await persistConfig(providers, [...draftModels], defaultProvider, updatedDefaultModel, 'Models');
        setEditingModels(false);
    };

    const handleAddDraftModel = () => {
        const provider = newModelProvider.trim();
        const model = newModelName.trim();
        const displayName = newModelDisplayName.trim();

        if (!provider || !model) {
            setError('Provider and model name are required');
            return;
        }

        if (!providers.some(p => p.name === provider)) {
            setError(`Provider "${provider}" does not exist. Save providers first.`);
            return;
        }

        if (draftModels.some(m => m.provider === provider && m.model === model)) {
            setError(`Model "${model}" for provider "${provider}" already exists`);
            return;
        }

        setDraftModels([...draftModels, { provider, model, display_name: displayName || undefined }]);
        setNewModelProvider('');
        setNewModelName('');
        setNewModelDisplayName('');
        setError(null);
    };

    const handleRemoveDraftModel = (index: number) => {
        setDraftModels(draftModels.filter((_, i) => i !== index));
    };

    const startEditDefaults = () => {
        setDraftDefaultProvider(defaultProvider);
        setDraftDefaultModel(defaultModel);
        setEditingDefaults(true);
        setError(null);
        setSuccess(null);
    };

    const cancelEditDefaults = () => {
        setEditingDefaults(false);
        setError(null);
    };

    const saveEditDefaults = async () => {
        await persistConfig(providers, models, draftDefaultProvider, draftDefaultModel, 'Defaults');
        setEditingDefaults(false);
    };

    const handleDraftDefaultProviderChange = (providerName: string) => {
        setDraftDefaultProvider(providerName);
        const providerModels = models.filter(m => m.provider === providerName);
        if (!providerModels.some(m => m.model === draftDefaultModel)) {
            setDraftDefaultModel(providerModels.length > 0 ? providerModels[0].model : '');
        }
    };

    return (
        <div className="diagnose-section">
            <h3 className="diagnose-section-title">AI Models</h3>

            {loading ? (
                <div className="diagnose-loading">Loading...</div>
            ) : (
                <div className="ai-models-section-content">
                    {!usingNewFile && (
                        <div className="ai-models-info">
                            Using legacy config from .config.local.json. Saving will migrate to new file (.ai-critic/ai-models.json).
                        </div>
                    )}

                    {/* Providers Section */}
                    <div className="ai-models-section-group">
                        <div className="ai-models-section-header">
                            <label className="ai-models-section-label">Providers</label>
                            {!editingProviders && (
                                <button className="ai-models-edit-btn" onClick={startEditProviders} title="Edit providers">
                                    <EditIcon />
                                </button>
                            )}
                        </div>
                        <p className="ai-models-section-desc">
                            Configure AI providers with their base URLs and API keys.
                        </p>

                        {!editingProviders ? (
                            providers.length > 0 && (
                                <div className="ai-models-list">
                                    {providers.map((p, i) => (
                                        <div key={i} className="ai-models-item">
                                            <div className="ai-models-item-content">
                                                <div className="ai-models-item-name">{p.name}</div>
                                                <div className="ai-models-item-details">{p.base_url}</div>
                                                {p.api_key && (
                                                    <div className="ai-models-item-details">API Key: {'*'.repeat(Math.min(p.api_key.length, 8))}</div>
                                                )}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )
                        ) : (
                            <>
                                {draftProviders.length > 0 && (
                                    <div className="ai-models-list">
                                        {draftProviders.map((p, i) => (
                                            <div key={i} className="ai-models-item ai-models-item-editable">
                                                <div className="ai-models-item-content">
                                                    <div className="ai-models-item-name">{p.name}</div>
                                                    <div className="ai-models-item-details">{p.base_url}</div>
                                                    {p.api_key && (
                                                        <div className="ai-models-item-details">API Key: {'*'.repeat(Math.min(p.api_key.length, 8))}</div>
                                                    )}
                                                </div>
                                                <button
                                                    className="ai-models-item-remove"
                                                    onClick={() => handleRemoveDraftProvider(i)}
                                                    title="Remove"
                                                >
                                                    ×
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}

                                <div className="ai-models-add-form">
                                    <FlexInput
                                        inputClassName="ai-models-input"
                                        value={newProviderName}
                                        onChange={setNewProviderName}
                                        placeholder="Provider name (e.g., openai)"
                                    />
                                    <FlexInput
                                        inputClassName="ai-models-input"
                                        value={newProviderBaseURL}
                                        onChange={setNewProviderBaseURL}
                                        placeholder="Base URL (e.g., https://api.openai.com/v1)"
                                    />
                                    <FlexInput
                                        inputClassName="ai-models-input"
                                        type="password"
                                        value={newProviderAPIKey}
                                        onChange={setNewProviderAPIKey}
                                        placeholder="API Key"
                                    />
                                    <button
                                        className="mcc-port-action-btn"
                                        onClick={handleAddDraftProvider}
                                        disabled={!newProviderName.trim() || !newProviderBaseURL.trim()}
                                    >
                                        Add Provider
                                    </button>
                                </div>

                                <div className="ai-models-edit-actions">
                                    <button className="ai-models-cancel-btn" onClick={cancelEditProviders}>Cancel</button>
                                    <button className="mcc-port-action-btn" onClick={saveEditProviders} disabled={saving === 'Providers'}>
                                        {saving === 'Providers' ? 'Saving...' : 'Save'}
                                    </button>
                                </div>
                            </>
                        )}
                    </div>

                    {/* Models Section */}
                    <div className="ai-models-section-group">
                        <div className="ai-models-section-header">
                            <label className="ai-models-section-label">Models</label>
                            {!editingModels && (
                                <button className="ai-models-edit-btn" onClick={startEditModels} title="Edit models">
                                    <EditIcon />
                                </button>
                            )}
                        </div>
                        <p className="ai-models-section-desc">
                            Configure models for each provider.
                        </p>

                        {!editingModels ? (
                            models.length > 0 && (
                                <div className="ai-models-list">
                                    {models.map((m, i) => (
                                        <div key={i} className="ai-models-item">
                                            <div className="ai-models-item-content">
                                                <div className="ai-models-item-name">
                                                    {m.display_name || m.model}
                                                    {m.display_name && <span className="ai-models-item-id"> ({m.model})</span>}
                                                </div>
                                                <div className="ai-models-item-details">Provider: {m.provider}</div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )
                        ) : (
                            <>
                                {draftModels.length > 0 && (
                                    <div className="ai-models-list">
                                        {draftModels.map((m, i) => (
                                            <div key={i} className="ai-models-item ai-models-item-editable">
                                                <div className="ai-models-item-content">
                                                    <div className="ai-models-item-name">
                                                        {m.display_name || m.model}
                                                        {m.display_name && <span className="ai-models-item-id"> ({m.model})</span>}
                                                    </div>
                                                    <div className="ai-models-item-details">Provider: {m.provider}</div>
                                                </div>
                                                <button
                                                    className="ai-models-item-remove"
                                                    onClick={() => handleRemoveDraftModel(i)}
                                                    title="Remove"
                                                >
                                                    ×
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}

                                <div className="ai-models-add-form">
                                    <select
                                        className="ai-models-select"
                                        value={newModelProvider}
                                        onChange={e => setNewModelProvider(e.target.value)}
                                    >
                                        <option value="">Select provider...</option>
                                        {providers.map(p => (
                                            <option key={p.name} value={p.name}>{p.name}</option>
                                        ))}
                                    </select>
                                    <FlexInput
                                        inputClassName="ai-models-input"
                                        value={newModelName}
                                        onChange={setNewModelName}
                                        placeholder="Model ID (e.g., gpt-4)"
                                    />
                                    <FlexInput
                                        inputClassName="ai-models-input"
                                        value={newModelDisplayName}
                                        onChange={setNewModelDisplayName}
                                        placeholder="Display name (optional)"
                                    />
                                    <button
                                        className="mcc-port-action-btn"
                                        onClick={handleAddDraftModel}
                                        disabled={!newModelProvider || !newModelName.trim()}
                                    >
                                        Add Model
                                    </button>
                                </div>

                                <div className="ai-models-edit-actions">
                                    <button className="ai-models-cancel-btn" onClick={cancelEditModels}>Cancel</button>
                                    <button className="mcc-port-action-btn" onClick={saveEditModels} disabled={saving === 'Models'}>
                                        {saving === 'Models' ? 'Saving...' : 'Save'}
                                    </button>
                                </div>
                            </>
                        )}
                    </div>

                    {/* Default Selection */}
                    <div className="ai-models-section-group">
                        <div className="ai-models-section-header">
                            <label className="ai-models-section-label">Defaults</label>
                            {!editingDefaults && (
                                <button className="ai-models-edit-btn" onClick={startEditDefaults} title="Edit defaults">
                                    <EditIcon />
                                </button>
                            )}
                        </div>
                        <p className="ai-models-section-desc">
                            Select the default provider and model for AI operations.
                        </p>

                        {!editingDefaults ? (
                            <>
                                <div className="ai-models-default-row">
                                    <label>Default Provider:</label>
                                    <span className="ai-models-default-value">{defaultProvider || <em>Not set</em>}</span>
                                </div>
                                <div className="ai-models-default-row">
                                    <label>Default Model:</label>
                                    <span className="ai-models-default-value">{defaultModel || <em>Not set</em>}</span>
                                </div>
                            </>
                        ) : (
                            <>
                                <div className="ai-models-default-row">
                                    <label>Default Provider:</label>
                                    <select
                                        className="ai-models-select"
                                        value={draftDefaultProvider}
                                        onChange={e => handleDraftDefaultProviderChange(e.target.value)}
                                    >
                                        <option value="">Select...</option>
                                        {providers.map(p => (
                                            <option key={p.name} value={p.name}>{p.name}</option>
                                        ))}
                                    </select>
                                </div>

                                <div className="ai-models-default-row">
                                    <label>Default Model:</label>
                                    <select
                                        className="ai-models-select"
                                        value={draftDefaultModel}
                                        onChange={e => setDraftDefaultModel(e.target.value)}
                                        disabled={!draftDefaultProvider}
                                    >
                                        <option value="">Select...</option>
                                        {models
                                            .filter(m => m.provider === draftDefaultProvider)
                                            .map(m => (
                                                <option key={m.model} value={m.model}>
                                                    {m.display_name || m.model}
                                                </option>
                                            ))}
                                    </select>
                                </div>

                                <div className="ai-models-edit-actions">
                                    <button className="ai-models-cancel-btn" onClick={cancelEditDefaults}>Cancel</button>
                                    <button className="mcc-port-action-btn" onClick={saveEditDefaults} disabled={saving === 'Defaults'}>
                                        {saving === 'Defaults' ? 'Saving...' : 'Save'}
                                    </button>
                                </div>
                            </>
                        )}
                    </div>

                    {error && <div className="ai-models-error">{error}</div>}
                    {success && <div className="ai-models-success">{success}</div>}
                </div>
            )}
        </div>
    );
}
