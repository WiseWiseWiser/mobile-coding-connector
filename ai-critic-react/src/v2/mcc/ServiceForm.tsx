import { useEffect, useState } from 'react';
import { fetchOwnedDomains } from '../../api/cloudflare';
import type { ServicePreset } from '../../api/services';
import type { ProviderInfo } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { PlusIcon } from '../../pure-view/icons/PlusIcon';
import { ServiceAuthFields } from './ServiceAuthFields';
import type { ServiceFormState } from './serviceFormState';

export interface ServiceFormProps {
    open: boolean;
    form: ServiceFormState;
    onChange: (updater: (prev: ServiceFormState) => ServiceFormState) => void;
    onSubmit: () => void;
    onCancel: () => void;
    onOpenNew: () => void;
    saving: boolean;
    homeDir: string;
    providers: ProviderInfo[];
    /** Presets prefill the form; nothing is created until the user saves. */
    presets: ServicePreset[];
    onApplyPreset: (preset: ServicePreset) => void;
}

/** ServiceForm adds or edits a user service, or offers to open the form. */
export function ServiceForm({ open, form, onChange, onSubmit, onCancel, onOpenNew, saving, homeDir, providers, presets, onApplyPreset }: ServiceFormProps) {
    if (!open) {
        return (
            <button className="mcc-add-port-btn" onClick={onOpenNew}>
                <PlusIcon />
                <span>Add Service</span>
            </button>
        );
    }

    return (
        <div className="mcc-add-port-form">
            <div className="mcc-add-port-header">
                <span>{form.id ? 'Edit Service' : 'Add Service'}</span>
                <button className="mcc-close-btn" onClick={onCancel}>×</button>
            </div>

            {!form.id && presets.length > 0 && (
                <div className="mcc-service-presets">
                    <label>Start from a preset</label>
                    <div className="mcc-service-preset-list">
                        {presets.map((preset) => (
                            <button
                                key={preset.id}
                                type="button"
                                className="mcc-service-preset-btn"
                                title={preset.description}
                                onClick={() => onApplyPreset(preset)}
                            >
                                {preset.name}
                            </button>
                        ))}
                    </div>
                </div>
            )}

            <div className="mcc-service-form-grid">
                <div className="mcc-form-field">
                    <label>Name</label>
                    <input
                        type="text"
                        placeholder="web"
                        value={form.name}
                        onChange={(e) => onChange((prev) => ({ ...prev, name: e.target.value }))}
                    />
                </div>
                <div className="mcc-form-field">
                    <label>Command</label>
                    <input
                        type="text"
                        placeholder="npm run dev"
                        value={form.command}
                        onChange={(e) => onChange((prev) => ({ ...prev, command: e.target.value }))}
                    />
                </div>
                <div className="mcc-form-field">
                    <label>Working Dir</label>
                    <input
                        type="text"
                        placeholder={homeDir || '/home/user'}
                        value={form.workingDir}
                        onChange={(e) => onChange((prev) => ({ ...prev, workingDir: e.target.value }))}
                    />
                </div>
                <div className="mcc-form-field">
                    <label>Extra Env</label>
                    <textarea
                        className="mcc-service-env-input"
                        placeholder={'NODE_ENV=development\nPORT=3000'}
                        value={form.extraEnvText}
                        onChange={(e) => onChange((prev) => ({ ...prev, extraEnvText: e.target.value }))}
                        rows={5}
                    />
                    <div className="mcc-service-field-hint">One <code>KEY=VALUE</code> entry per line.</div>
                </div>
            </div>

            <button
                type="button"
                className={`mcc-service-toggle ${form.enablePortForward ? 'active' : ''}`}
                onClick={() => onChange((prev) => ({ ...prev, enablePortForward: !prev.enablePortForward }))}
            >
                {form.enablePortForward ? 'Port forwarding enabled' : 'Add port forwarding'}
            </button>

            {form.enablePortForward && (
                <>
                    <div className="mcc-add-port-fields">
                        <div className="mcc-form-field">
                            <label>Service Port</label>
                            <input
                                type="number"
                                placeholder="3000"
                                value={form.port}
                                onChange={(e) => onChange((prev) => ({ ...prev, port: e.target.value }))}
                            />
                        </div>
                        <div className="mcc-form-field">
                            <label>Forward Label</label>
                            <input
                                type="text"
                                placeholder="app.example.com"
                                value={form.label}
                                onChange={(e) => onChange((prev) => ({ ...prev, label: e.target.value }))}
                            />
                        </div>
                    </div>
                    <div className="mcc-form-field mcc-provider-field">
                        <label>Provider</label>
                        <div className="mcc-provider-options">
                            {providers.map((provider) => (
                                <button
                                    key={provider.id}
                                    className={`mcc-provider-btn ${form.provider === provider.id ? 'active' : ''}`}
                                    onClick={() => onChange((prev) => ({ ...prev, provider: provider.id as ServiceFormState['provider'] }))}
                                    title={provider.description}
                                    type="button"
                                >
                                    {provider.name}
                                </button>
                            ))}
                        </div>
                    </div>
                    {(form.provider === TunnelProviders.CloudflareTunnel || form.provider === TunnelProviders.CloudflareOwned) && (
                        <>
                            <OwnedDomainsPicker
                                selectedDomain={form.baseDomain}
                                onSelect={(domain) => onChange((prev) => ({ ...prev, baseDomain: domain }))}
                            />
                            <div className="mcc-form-field mcc-subdomain-field">
                                <label>Subdomain</label>
                                <input
                                    type="text"
                                    placeholder="brave-apex-dawn"
                                    value={form.subdomain}
                                    onChange={(e) => onChange((prev) => ({ ...prev, subdomain: e.target.value }))}
                                    className="mcc-subdomain-input"
                                />
                            </div>
                        </>
                    )}
                </>
            )}

            <ServiceAuthFields
                requireAuth={form.requireAuth}
                authUserMode={form.authUserMode}
                authUser={form.authUser}
                authTokenMode={form.authTokenMode}
                authToken={form.authToken}
                onRequireAuthChange={(requireAuth) => onChange((prev) => ({ ...prev, requireAuth }))}
                onAuthUserModeChange={(authUserMode) => onChange((prev) => ({ ...prev, authUserMode }))}
                onAuthUserChange={(authUser) => onChange((prev) => ({ ...prev, authUser }))}
                onAuthTokenModeChange={(authTokenMode) => onChange((prev) => ({ ...prev, authTokenMode }))}
                onAuthTokenChange={(authToken) => onChange((prev) => ({ ...prev, authToken }))}
            />

            <button className="mcc-forward-btn" onClick={onSubmit} disabled={saving}>
                {saving ? 'Saving...' : form.id ? 'Update Service' : 'Save Service'}
            </button>
        </div>
    );
}

interface OwnedDomainsPickerProps {
    selectedDomain: string;
    onSelect: (domain: string) => void;
}

function OwnedDomainsPicker({ selectedDomain, onSelect }: OwnedDomainsPickerProps) {
    const [domains, setDomains] = useState<string[]>([]);

    useEffect(() => {
        fetchOwnedDomains()
            .then((items) => {
                setDomains(items);
                if (items.length > 0 && !selectedDomain) {
                    onSelect(items[0]);
                }
            })
            .catch(() => {});
    }, []);

    if (domains.length === 0) return null;

    return (
        <div className="mcc-owned-domains-hint">
            <label>Base Domain</label>
            <div className="mcc-owned-domains-list">
                {domains.map((domain) => (
                    <button
                        key={domain}
                        className={`mcc-owned-domain-btn ${selectedDomain === domain ? 'active' : ''}`}
                        onClick={() => onSelect(domain)}
                        type="button"
                    >
                        {domain}
                    </button>
                ))}
            </div>
        </div>
    );
}
