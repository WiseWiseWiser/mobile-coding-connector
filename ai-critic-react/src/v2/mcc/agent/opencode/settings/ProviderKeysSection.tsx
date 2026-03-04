import { useState, useRef, useEffect, useMemo } from 'react';
import type { OpencodeAuthKeyEntry, WellKnownProvider } from '../../../../../api/agents';

export interface ProviderKeysSectionProps {
    authKeys: OpencodeAuthKeyEntry[];
    wellKnownProviders: WellKnownProvider[];
    onSaveKey: (provider: string, key: string, baseUrl?: string) => Promise<void>;
    onDeleteKey: (provider: string) => Promise<void>;
}

const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 10px',
    background: '#0f172a',
    border: '1px solid #475569',
    borderRadius: 6,
    color: '#e2e8f0',
    fontSize: '13px',
    boxSizing: 'border-box',
};

const smallBtnStyle = (color: string): React.CSSProperties => ({
    padding: '4px 10px',
    fontSize: '12px',
    background: 'transparent',
    border: '1px solid #475569',
    borderRadius: 4,
    color,
    cursor: 'pointer',
});

export function ProviderKeysSection({ authKeys, wellKnownProviders, onSaveKey, onDeleteKey }: ProviderKeysSectionProps) {
    const [editingProvider, setEditingProvider] = useState<string | null>(null);
    const [editingKey, setEditingKey] = useState('');
    const [editingBaseUrl, setEditingBaseUrl] = useState('');
    const [showAddForm, setShowAddForm] = useState(false);
    const [newProvider, setNewProvider] = useState('');
    const [newKey, setNewKey] = useState('');
    const [newBaseUrl, setNewBaseUrl] = useState('');
    const [savingKey, setSavingKey] = useState(false);
    const [error, setError] = useState('');
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    const providerBaseUrlMap = useMemo(() => {
        const map = new Map<string, string>();
        for (const p of wellKnownProviders) {
            map.set(p.name, p.base_url);
        }
        return map;
    }, [wellKnownProviders]);

    useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                setDropdownOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const handleSaveKey = async (provider: string, key: string, baseUrl: string) => {
        setSavingKey(true);
        setError('');
        try {
            await onSaveKey(provider, key, baseUrl || undefined);
            setEditingProvider(null);
            setEditingKey('');
            setEditingBaseUrl('');
            setShowAddForm(false);
            setNewProvider('');
            setNewKey('');
            setNewBaseUrl('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save key');
        } finally {
            setSavingKey(false);
        }
    };

    const handleDeleteKey = async (provider: string) => {
        setSavingKey(true);
        setError('');
        try {
            await onDeleteKey(provider);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete key');
        } finally {
            setSavingKey(false);
        }
    };

    const startEditing = (entry: OpencodeAuthKeyEntry) => {
        setEditingProvider(entry.provider);
        setEditingKey('');
        setEditingBaseUrl(entry.base_url || providerBaseUrlMap.get(entry.provider) || '');
    };

    const cancelEditing = () => {
        setEditingProvider(null);
        setEditingKey('');
        setEditingBaseUrl('');
    };

    const selectNewProvider = (name: string) => {
        setNewProvider(name);
        setNewBaseUrl(providerBaseUrlMap.get(name) || '');
        setDropdownOpen(false);
    };

    const handleNewProviderChange = (value: string) => {
        setNewProvider(value);
        const defaultUrl = providerBaseUrlMap.get(value);
        if (defaultUrl) {
            setNewBaseUrl(defaultUrl);
        }
        setDropdownOpen(true);
    };

    const existingProviders = new Set(authKeys.map(e => e.provider));
    const availableProviders = wellKnownProviders
        .map(p => p.name)
        .filter(name => !existingProviders.has(name));
    const filteredProviders = newProvider
        ? availableProviders.filter(p => p.toLowerCase().includes(newProvider.toLowerCase()))
        : availableProviders;

    return (
        <div className="mcc-agent-settings-field" style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
            <label className="mcc-agent-settings-label">
                Provider API Keys
            </label>
            <div className="mcc-agent-settings-hint" style={{ marginBottom: 12, fontSize: '13px', color: '#94a3b8' }}>
                Configure API keys for LLM providers (stored in ~/.local/share/opencode/auth.json).
            </div>

            {error && (
                <div style={{ marginBottom: 8, padding: '6px 10px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 4, color: '#fca5a5', fontSize: '12px' }}>
                    {error}
                </div>
            )}

            {authKeys.map(entry => (
                <div key={entry.provider} style={{
                    marginBottom: 8,
                    padding: '10px 12px',
                    background: '#1e293b',
                    border: '1px solid #334155',
                    borderRadius: 8,
                }}>
                    {editingProvider === entry.provider ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                            <strong style={{ color: '#e2e8f0' }}>{entry.provider}</strong>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                                <label style={{ fontSize: '11px', color: '#94a3b8' }}>Base URL</label>
                                <input
                                    type="text"
                                    value={editingBaseUrl}
                                    onChange={e => setEditingBaseUrl(e.target.value)}
                                    placeholder="https://api.example.com/v1"
                                    disabled={savingKey}
                                    style={inputStyle}
                                />
                            </div>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                                <label style={{ fontSize: '11px', color: '#94a3b8' }}>API Key</label>
                                <input
                                    type="password"
                                    value={editingKey}
                                    onChange={e => setEditingKey(e.target.value)}
                                    placeholder="Enter new API key..."
                                    disabled={savingKey}
                                    style={inputStyle}
                                />
                            </div>
                            <div style={{ display: 'flex', gap: 8 }}>
                                <button
                                    onClick={() => handleSaveKey(entry.provider, editingKey, editingBaseUrl)}
                                    disabled={savingKey || !editingKey.trim()}
                                    style={{ padding: '4px 10px', fontSize: '12px', background: '#22c55e', border: 'none', borderRadius: 4, color: '#fff', cursor: 'pointer' }}
                                >
                                    Save
                                </button>
                                <button onClick={cancelEditing} disabled={savingKey} style={smallBtnStyle('#94a3b8')}>
                                    Cancel
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                <strong style={{ color: '#e2e8f0', minWidth: 100 }}>{entry.provider}</strong>
                                <div style={{ marginLeft: 'auto', display: 'flex', gap: 6 }}>
                                    <button onClick={() => startEditing(entry)} style={smallBtnStyle('#60a5fa')}>
                                        Edit
                                    </button>
                                    <button onClick={() => handleDeleteKey(entry.provider)} disabled={savingKey} style={smallBtnStyle('#f87171')}>
                                        Delete
                                    </button>
                                </div>
                            </div>
                            {entry.base_url && (
                                <span style={{ fontSize: '12px', color: '#64748b', fontFamily: 'monospace' }}>{entry.base_url}</span>
                            )}
                            <span style={{ fontFamily: 'monospace', fontSize: '12px', color: '#94a3b8' }}>
                                {entry.masked_key || '(empty)'}
                            </span>
                        </div>
                    )}
                </div>
            ))}

            {!showAddForm ? (
                <button
                    onClick={() => setShowAddForm(true)}
                    style={{
                        marginTop: authKeys.length > 0 ? 12 : 0,
                        padding: '8px 16px',
                        fontSize: '13px',
                        background: '#3b82f6',
                        border: 'none',
                        borderRadius: 6,
                        color: '#fff',
                        fontWeight: 500,
                        cursor: 'pointer',
                        alignSelf: 'flex-start',
                    }}
                >
                    + Add Provider
                </button>
            ) : (
                <div style={{
                    marginTop: authKeys.length > 0 ? 12 : 0,
                    padding: '12px',
                    background: '#1e293b',
                    border: '1px solid #334155',
                    borderRadius: 8,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 10,
                }}>
                    {/* Provider row */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        <label style={{ fontSize: '12px', color: '#94a3b8', fontWeight: 500 }}>Provider</label>
                        <div ref={dropdownRef} style={{ position: 'relative' }}>
                            <input
                                type="text"
                                value={newProvider}
                                onChange={e => handleNewProviderChange(e.target.value)}
                                onFocus={() => setDropdownOpen(true)}
                                placeholder="Select or type provider name..."
                                disabled={savingKey}
                                style={inputStyle}
                            />
                            {dropdownOpen && filteredProviders.length > 0 && (
                                <div style={{
                                    position: 'absolute',
                                    top: '100%',
                                    left: 0,
                                    right: 0,
                                    marginTop: 2,
                                    background: '#1e293b',
                                    border: '1px solid #475569',
                                    borderRadius: 6,
                                    maxHeight: 180,
                                    overflowY: 'auto',
                                    zIndex: 10,
                                }}>
                                    {filteredProviders.map(p => (
                                        <div
                                            key={p}
                                            onClick={() => selectNewProvider(p)}
                                            style={{
                                                padding: '6px 10px',
                                                cursor: 'pointer',
                                                fontSize: '13px',
                                                color: '#e2e8f0',
                                                display: 'flex',
                                                justifyContent: 'space-between',
                                                alignItems: 'center',
                                                borderBottom: '1px solid #334155',
                                            }}
                                            onMouseEnter={e => (e.currentTarget.style.background = 'rgba(59,130,246,0.15)')}
                                            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                                        >
                                            <span>{p}</span>
                                            <span style={{ fontSize: '11px', color: '#64748b', fontFamily: 'monospace' }}>
                                                {providerBaseUrlMap.get(p)}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Base URL row */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        <label style={{ fontSize: '12px', color: '#94a3b8', fontWeight: 500 }}>Base URL</label>
                        <input
                            type="text"
                            value={newBaseUrl}
                            onChange={e => setNewBaseUrl(e.target.value)}
                            placeholder="https://api.example.com/v1"
                            disabled={savingKey}
                            style={inputStyle}
                        />
                    </div>

                    {/* API key row */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        <label style={{ fontSize: '12px', color: '#94a3b8', fontWeight: 500 }}>API Key</label>
                        <input
                            type="password"
                            value={newKey}
                            onChange={e => setNewKey(e.target.value)}
                            placeholder="Enter API key..."
                            disabled={savingKey}
                            style={inputStyle}
                        />
                    </div>

                    {/* Actions */}
                    <div style={{ display: 'flex', gap: 8 }}>
                        <button
                            onClick={() => handleSaveKey(newProvider.trim(), newKey, newBaseUrl.trim())}
                            disabled={savingKey || !newProvider.trim() || !newKey.trim()}
                            style={{
                                padding: '8px 16px',
                                fontSize: '13px',
                                background: newProvider.trim() && newKey.trim() ? '#22c55e' : '#475569',
                                border: 'none',
                                borderRadius: 6,
                                color: '#fff',
                                fontWeight: 500,
                                cursor: (savingKey || !newProvider.trim() || !newKey.trim()) ? 'not-allowed' : 'pointer',
                                opacity: (savingKey || !newProvider.trim() || !newKey.trim()) ? 0.6 : 1,
                            }}
                        >
                            {savingKey ? 'Saving...' : 'Save'}
                        </button>
                        <button
                            onClick={() => { setShowAddForm(false); setNewProvider(''); setNewKey(''); setNewBaseUrl(''); setDropdownOpen(false); }}
                            disabled={savingKey}
                            style={{
                                padding: '8px 16px',
                                fontSize: '13px',
                                background: 'transparent',
                                border: '1px solid #475569',
                                borderRadius: 6,
                                color: '#94a3b8',
                                cursor: 'pointer',
                            }}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
