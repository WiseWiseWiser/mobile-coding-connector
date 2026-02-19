import { useState } from 'react';
import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import { FlexInput } from '../../../../pure-view/FlexInput';

export interface AddDomainFormProps {
    onAdd: (entry: DomainEntry) => void;
    onCancel: () => void;
    onError: (err: string | null) => void;
}

export function AddDomainForm({ onAdd, onCancel, onError }: AddDomainFormProps) {
    const [domain, setDomain] = useState('');
    const [provider, setProvider] = useState<string>(DomainProviders.Cloudflare);

    const handleAdd = () => {
        if (!domain.trim()) return;
        onAdd({ domain: domain.trim(), provider });
    };

    return (
        <div className="diagnose-webaccess-add">
            <div className="diagnose-webaccess-add-row">
                <label className="diagnose-webaccess-add-label">Domain</label>
                <FlexInput
                    inputClassName="diagnose-webaccess-input"
                    placeholder="e.g. myapp.example.com"
                    value={domain}
                    onChange={setDomain}
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
            <div className="diagnose-webaccess-add-actions">
                <button className="diagnose-webaccess-add-btn" onClick={handleAdd} disabled={!domain.trim()}>
                    Add Domain
                </button>
                <button className="diagnose-webaccess-cancel-btn" onClick={onCancel}>
                    Cancel
                </button>
            </div>
        </div>
    );
}
