import { useState } from 'react';
import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';

export interface DomainRowEditProps {
    entry: DomainEntry;
    tunnelName: string;
    onSave: (entry: DomainEntry, tunnelName: string) => void;
    onRemove: () => void;
    onCancel: () => void;
    onError: (err: string | null) => void;
}

export function DomainRowEdit({ entry, tunnelName, onSave, onRemove, onCancel, onError }: DomainRowEditProps) {
    const [domain, setDomain] = useState(entry.domain);
    const [provider, setProvider] = useState(entry.provider);
    const [editTunnelName, setEditTunnelName] = useState(tunnelName);

    const handleSave = () => {
        if (!domain.trim()) return;
        onSave({ domain: domain.trim(), provider }, editTunnelName);
    };

    return (
        <div className="diagnose-webaccess-row diagnose-webaccess-row--editing">
            <div className="diagnose-webaccess-add-row">
                <label className="diagnose-webaccess-add-label">Domain</label>
                <input
                    type="text"
                    className="diagnose-webaccess-input"
                    placeholder="e.g. myapp.example.com"
                    value={domain}
                    onChange={e => setDomain(e.target.value)}
                />
                <button
                    type="button"
                    className="diagnose-webaccess-generate-btn"
                    onClick={async () => {
                        try {
                            const d = await fetchRandomDomain(domain || undefined);
                            setDomain(d);
                        } catch (err) {
                            onError(err instanceof Error ? err.message : String(err));
                        }
                    }}
                >
                    Generate Random Subdomain
                </button>
            </div>
            <div className="diagnose-webaccess-add-row">
                <label className="diagnose-webaccess-add-label">Provider</label>
                <select
                    className="diagnose-webaccess-select"
                    value={provider}
                    onChange={e => setProvider(e.target.value)}
                >
                    <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                    <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                </select>
            </div>
            {provider === DomainProviders.Cloudflare && (
                <div className="diagnose-webaccess-add-row">
                    <label className="diagnose-webaccess-add-label">Cloudflare Tunnel Name</label>
                    <input
                        type="text"
                        className="diagnose-webaccess-input"
                        placeholder="auto (derived from domain)"
                        value={editTunnelName}
                        onChange={e => setEditTunnelName(e.target.value)}
                    />
                    <span className="diagnose-webaccess-tunnel-name-hint">
                        Shared Cloudflare named tunnel identifier. Default: auto (derived from domain)
                    </span>
                </div>
            )}
            <div className="diagnose-webaccess-edit-actions">
                <button className="diagnose-webaccess-add-btn" onClick={handleSave} disabled={!domain.trim()}>
                    Save
                </button>
                <button className="diagnose-webaccess-cancel-btn" onClick={onCancel}>Cancel</button>
                <button className="diagnose-webaccess-remove-btn" onClick={onRemove}>Remove</button>
            </div>
        </div>
    );
}
