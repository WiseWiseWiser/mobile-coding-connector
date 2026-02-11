import { useState, useEffect } from 'react';
import { fetchAIConfig, saveAIConfig, type AIProvider, type AIModel } from '../../../../api/ai';
import './AIModelsSection.css';

export function AIModelsSection() {
    const [providers, setProviders] = useState<AIProvider[]>([]);
    const [models, setModels] = useState<AIModel[]>([]);
    const [defaultProvider, setDefaultProvider] = useState('');
    const [defaultModel, setDefaultModel] = useState('');
    const [usingNewFile, setUsingNewFile] = useState(false);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);

    // Form states for adding new provider
    const [newProviderName, setNewProviderName] = useState('');
    const [newProviderBaseURL, setNewProviderBaseURL] = useState('');
    const [newProviderAPIKey, setNewProviderAPIKey] = useState('');

    // Form states for adding new model
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
                setUsingNewFile(cfg.using_new_file);
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setSuccess(false);
        try {
            await saveAIConfig({
                providers,
                models,
                default_provider: defaultProvider,
                default_model: defaultModel,
            });
            setSuccess(true);
            setUsingNewFile(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setSaving(false);
    };

    const handleAddProvider = () => {
        const name = newProviderName.trim();
        const baseURL = newProviderBaseURL.trim();
        const apiKey = newProviderAPIKey.trim();

        if (!name || !baseURL) {
            setError('Provider name and base URL are required');
            return;
        }

        if (providers.some(p => p.name === name)) {
            setError(`Provider "${name}" already exists`);
            return;
        }

        setProviders([...providers, { name, base_url: baseURL, api_key: apiKey }]);
        setNewProviderName('');
        setNewProviderBaseURL('');
        setNewProviderAPIKey('');
        setError(null);
        setSuccess(false);
    };

    const handleRemoveProvider = (index: number) => {
        const provider = providers[index];
        setProviders(providers.filter((_, i) => i !== index));
        // Also remove models associated with this provider
        setModels(models.filter(m => m.provider !== provider.name));
        // Update default if needed
        if (defaultProvider === provider.name) {
            setDefaultProvider('');
            setDefaultModel('');
        }
        setSuccess(false);
    };

    const handleAddModel = () => {
        const provider = newModelProvider.trim();
        const model = newModelName.trim();
        const displayName = newModelDisplayName.trim();

        if (!provider || !model) {
            setError('Provider and model name are required');
            return;
        }

        if (!providers.some(p => p.name === provider)) {
            setError(`Provider "${provider}" does not exist`);
            return;
        }

        if (models.some(m => m.provider === provider && m.model === model)) {
            setError(`Model "${model}" for provider "${provider}" already exists`);
            return;
        }

        setModels([...models, { provider, model, display_name: displayName || undefined }]);
        setNewModelProvider('');
        setNewModelName('');
        setNewModelDisplayName('');
        setError(null);
        setSuccess(false);
    };

    const handleRemoveModel = (index: number) => {
        const model = models[index];
        setModels(models.filter((_, i) => i !== index));
        // Update default if needed
        if (defaultModel === model.model && defaultProvider === model.provider) {
            setDefaultModel('');
        }
        setSuccess(false);
    };

    const handleSetDefaultProvider = (providerName: string) => {
        setDefaultProvider(providerName);
        // Only keep default model if it belongs to this provider
        const providerModels = models.filter(m => m.provider === providerName);
        if (!providerModels.some(m => m.model === defaultModel)) {
            setDefaultModel(providerModels.length > 0 ? providerModels[0].model : '');
        }
        setSuccess(false);
    };

    const handleSetDefaultModel = (modelName: string) => {
        setDefaultModel(modelName);
        setSuccess(false);
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
                        <label className="ai-models-section-label">Providers</label>
                        <p className="ai-models-section-desc">
                            Configure AI providers with their base URLs and API keys.
                        </p>

                        {providers.length > 0 && (
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
                                        <button
                                            className="ai-models-item-remove"
                                            onClick={() => handleRemoveProvider(i)}
                                            title="Remove"
                                        >
                                            ×
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}

                        <div className="ai-models-add-form">
                            <input
                                className="ai-models-input"
                                type="text"
                                value={newProviderName}
                                onChange={e => setNewProviderName(e.target.value)}
                                placeholder="Provider name (e.g., openai)"
                            />
                            <input
                                className="ai-models-input"
                                type="text"
                                value={newProviderBaseURL}
                                onChange={e => setNewProviderBaseURL(e.target.value)}
                                placeholder="Base URL (e.g., https://api.openai.com/v1)"
                            />
                            <input
                                className="ai-models-input"
                                type="password"
                                value={newProviderAPIKey}
                                onChange={e => setNewProviderAPIKey(e.target.value)}
                                placeholder="API Key"
                            />
                            <button
                                className="mcc-port-action-btn"
                                onClick={handleAddProvider}
                                disabled={!newProviderName.trim() || !newProviderBaseURL.trim()}
                            >
                                Add Provider
                            </button>
                        </div>
                    </div>

                    {/* Models Section */}
                    <div className="ai-models-section-group">
                        <label className="ai-models-section-label">Models</label>
                        <p className="ai-models-section-desc">
                            Configure models for each provider.
                        </p>

                        {models.length > 0 && (
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
                                        <button
                                            className="ai-models-item-remove"
                                            onClick={() => handleRemoveModel(i)}
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
                            <input
                                className="ai-models-input"
                                type="text"
                                value={newModelName}
                                onChange={e => setNewModelName(e.target.value)}
                                placeholder="Model ID (e.g., gpt-4)"
                            />
                            <input
                                className="ai-models-input"
                                type="text"
                                value={newModelDisplayName}
                                onChange={e => setNewModelDisplayName(e.target.value)}
                                placeholder="Display name (optional)"
                            />
                            <button
                                className="mcc-port-action-btn"
                                onClick={handleAddModel}
                                disabled={!newModelProvider || !newModelName.trim()}
                            >
                                Add Model
                            </button>
                        </div>
                    </div>

                    {/* Default Selection */}
                    <div className="ai-models-section-group">
                        <label className="ai-models-section-label">Defaults</label>
                        <p className="ai-models-section-desc">
                            Select the default provider and model for AI operations.
                        </p>

                        <div className="ai-models-default-row">
                            <label>Default Provider:</label>
                            <select
                                className="ai-models-select"
                                value={defaultProvider}
                                onChange={e => handleSetDefaultProvider(e.target.value)}
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
                                value={defaultModel}
                                onChange={e => handleSetDefaultModel(e.target.value)}
                                disabled={!defaultProvider}
                            >
                                <option value="">Select...</option>
                                {models
                                    .filter(m => m.provider === defaultProvider)
                                    .map(m => (
                                        <option key={m.model} value={m.model}>
                                            {m.display_name || m.model}
                                        </option>
                                    ))}
                            </select>
                        </div>
                    </div>

                    {error && <div className="ai-models-error">{error}</div>}
                    {success && <div className="ai-models-success">Saved successfully!</div>}

                    <div className="ai-models-actions">
                        <button
                            className="mcc-port-action-btn"
                            onClick={handleSave}
                            disabled={saving}
                        >
                            {saving ? 'Saving...' : 'Save'}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
