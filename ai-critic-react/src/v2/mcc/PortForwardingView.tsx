import { useState, useEffect } from 'react';
import type { TunnelProvider, ProviderInfo } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { fetchOwnedDomains } from '../../api/cloudflare';
import { PlusIcon } from '../../pure-view/icons/PlusIcon';
import { LocalPortsTable } from './LocalPortsTable';
import type { LocalPortInfo } from '../../api/ports';
import type { PortForward } from '../../hooks/usePortForwards';
import { CloudflareInlineDiagnostics } from './CloudflareInlineDiagnostics';
import { TunnelGroupsSection } from './TunnelGroupsSection';
import { PortForwardCard } from './PortForwardCard';

// ---- Port Forwarding View ----

export interface PortForwardingViewProps {
    ports: PortForward[];
    availableProviders: ProviderInfo[];
    loading: boolean;
    error: string | null;
    actionError: string | null;
    showNewForm: boolean;
    onToggleNewForm: () => void;
    newPortNumber: string;
    newPortLabel: string;
    newPortProvider: TunnelProvider;
    newPortSubdomain?: string;
    newPortBaseDomain?: string;
    onPortNumberChange: (value: string) => void;
    onPortLabelChange: (value: string) => void;
    onPortProviderChange: (value: TunnelProvider) => void;
    onPortSubdomainChange?: (value: string) => void;
    onPortBaseDomainChange?: (value: string) => void;
    onGenerateSubdomain?: () => void;
    onAddPort: () => void;
    onRemovePort: (port: number) => void;
    onNavigateToView: (view: string) => void;
    localPorts: LocalPortInfo[];
    localPortsLoading: boolean;
    localPortsError: string | null;
    onForwardLocalPort: (port: number) => void;
}

export function PortForwardingView({
    ports,
    availableProviders,
    loading,
    error,
    actionError,
    showNewForm,
    onToggleNewForm,
    newPortNumber,
    newPortLabel,
    newPortProvider,
    newPortSubdomain,
    newPortBaseDomain,
    onPortNumberChange,
    onPortLabelChange,
    onPortProviderChange,
    onPortSubdomainChange,
    onPortBaseDomainChange,
    onGenerateSubdomain,
    onAddPort,
    onRemovePort,
    onNavigateToView,
    localPorts,
    localPortsLoading,
    localPortsError,
    onForwardLocalPort,
}: PortForwardingViewProps) {
    return (
        <div className="mcc-ports">
            <div className="mcc-section-header">
                <h2>Port Forwarding</h2>
            </div>
            <CloudflareInlineDiagnostics />
            <TunnelGroupsSection />
            {error && (
                <div className="mcc-ports-error">Error: {error}</div>
            )}
            {actionError && (
                <div className="mcc-ports-error">{actionError}</div>
            )}

            <div className="mcc-ports-subtitle">
                {loading ? 'Loading...' : `Active Forwards (${ports.length})`}
            </div>
            <div className="mcc-ports-list">
                {ports.map(port => (
                    <PortForwardCard key={port.localPort} port={port} onRemove={() => onRemovePort(port.localPort)} onNavigateToView={onNavigateToView} />
                ))}
                {!loading && ports.length === 0 && (
                    <div className="mcc-ports-empty">No port forwards active.</div>
                )}
            </div>
            <div className="mcc-add-port-section">
                {showNewForm ? (
                    <div className="mcc-add-port-form">
                        <div className="mcc-add-port-header">
                            <span>Add Port Forward</span>
                            <button className="mcc-close-btn" onClick={onToggleNewForm}>×</button>
                        </div>
                        <div className="mcc-add-port-fields">
                            <div className="mcc-form-field">
                                <label>Port</label>
                                <input
                                    type="number"
                                    placeholder="8080"
                                    value={newPortNumber}
                                    onChange={e => onPortNumberChange(e.target.value)}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Label</label>
                                <input
                                    type="text"
                                    placeholder="My Service"
                                    value={newPortLabel}
                                    onChange={e => onPortLabelChange(e.target.value)}
                                />
                            </div>
                        </div>
                        <div className="mcc-form-field mcc-provider-field">
                            <label>Provider</label>
                            <div className="mcc-provider-options">
                                {availableProviders.filter(p => p.available).map(p => (
                                    <button
                                        key={p.id}
                                        className={`mcc-provider-btn ${newPortProvider === p.id ? 'active' : ''}`}
                                        onClick={() => onPortProviderChange(p.id as TunnelProvider)}
                                        title={p.description}
                                    >
                                        {p.name}
                                    </button>
                                ))}
                            </div>
                        </div>
                        {(newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned) && (
                            <>
                                <OwnedDomainsHint
                                    selectedDomain={newPortBaseDomain}
                                    onSelectDomain={onPortBaseDomainChange}
                                />
                                <div className="mcc-form-field mcc-subdomain-field">
                                    <label>Subdomain</label>
                                    <div className="mcc-subdomain-input-group">
                                        <input
                                            type="text"
                                            placeholder="brave-apex-dawn"
                                            value={newPortSubdomain || ''}
                                            onChange={e => onPortSubdomainChange?.(e.target.value)}
                                            className="mcc-subdomain-input"
                                        />
                                        <button
                                            className="mcc-generate-btn"
                                            onClick={onGenerateSubdomain}
                                            title="Generate random subdomain"
                                            type="button"
                                        >
                                            🎲
                                        </button>
                                    </div>
                                </div>
                                {newPortSubdomain && newPortBaseDomain && (
                                    <div className="mcc-domain-preview">
                                        <label>Full Domain</label>
                                        <div className="mcc-domain-preview-value">
                                            {newPortSubdomain}.{newPortBaseDomain}
                                        </div>
                                    </div>
                                )}
                            </>
                        )}
                        <button 
                            className="mcc-forward-btn" 
                            onClick={onAddPort}
                            disabled={!newPortNumber || ((newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned) && !newPortBaseDomain)}
                        >
                            Forward
                        </button>
                    </div>
                ) : (
                    <button className="mcc-add-port-btn" onClick={onToggleNewForm}>
                        <PlusIcon />
                        <span>Add Port Forward</span>
                    </button>
                )}
            </div>

            <LocalPortsTable
                ports={localPorts}
                loading={localPortsLoading}
                error={localPortsError}
                forwardedPorts={new Set(ports.map(p => p.localPort))}
                onForwardPort={onForwardLocalPort}
            />

            <PortForwardingHelp />
        </div>
    );
}

// ---- Owned Domains Hint ----

interface OwnedDomainsHintProps {
    selectedDomain?: string;
    onSelectDomain?: (value: string) => void;
}

function OwnedDomainsHint({ selectedDomain, onSelectDomain }: OwnedDomainsHintProps) {
    const [domains, setDomains] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        fetchOwnedDomains().then((domains) => {
            setDomains(domains);
            if (domains.length > 0 && !selectedDomain) {
                onSelectDomain?.(domains[0]);
            }
        }).catch(() => {}).finally(() => setLoading(false));
    }, []);

    if (loading) {
        return (
            <div className="mcc-owned-domains-hint">
                <label>Base Domain</label>
                <div className="mcc-owned-domains-loading">Loading domains...</div>
            </div>
        );
    }

    if (domains.length === 0) return null;

    return (
        <div className="mcc-owned-domains-hint">
            <label>Base Domain</label>
            <div className="mcc-owned-domains-list">
                {domains.map(d => (
                    <button
                        key={d}
                        className={`mcc-owned-domain-btn ${selectedDomain === d ? 'active' : ''}`}
                        onClick={() => onSelectDomain?.(d)}
                        title={`Use domain: ${d}`}
                        type="button"
                    >
                        {d}
                    </button>
                ))}
            </div>
            <div className="mcc-owned-domains-note">Select a domain as the base for your subdomain.</div>
        </div>
    );
}

// ---- Help section ----

function PortForwardingHelp() {
    const [expanded, setExpanded] = useState(false);

    return (
        <div className="mcc-ports-help">
            <button className="mcc-ports-help-toggle" onClick={() => setExpanded(!expanded)}>
                <span className="mcc-ports-help-icon">?</span>
                <span>How does port forwarding work?</span>
                <span className={`mcc-ports-help-chevron ${expanded ? 'expanded' : ''}`}>›</span>
            </button>
            {expanded && (
                <div className="mcc-ports-help-content">
                    <p>
                        Port forwarding creates a secure public URL for a service running on a local port
                        of this machine, making it accessible from anywhere on the internet.
                    </p>
                    <div className="mcc-ports-help-steps">
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">1</span>
                            <span>You specify a local port (e.g. <code>3000</code>) where your app is running.</span>
                        </div>
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">2</span>
                            <span>The server starts a tunnel process to create a public URL.</span>
                        </div>
                        <div className="mcc-ports-help-step">
                            <span className="mcc-ports-help-step-num">3</span>
                            <span>A temporary public URL is assigned that proxies traffic to your local service.</span>
                        </div>
                    </div>

                    <p><strong>Providers:</strong></p>
                    <div className="mcc-ports-help-provider">
                        <strong>localtunnel</strong> (default)
                        <div className="mcc-ports-help-cmd">
                            <code>npx localtunnel --port PORT</code>
                        </div>
                        <span>Free, no account needed. URL: <code>https://xxx.loca.lt</code></span>
                    </div>
                    <div className="mcc-ports-help-provider">
                        <strong>Cloudflare Quick Tunnel</strong>
                        <div className="mcc-ports-help-cmd">
                            <code>cloudflared tunnel --url http://127.0.0.1:PORT</code>
                        </div>
                        <span>Free via Cloudflare Quick Tunnels. URL: <code>https://xxx.trycloudflare.com</code></span>
                    </div>
                    <div className="mcc-ports-help-provider">
                        <strong>Cloudflare Named Tunnel</strong>
                        <div className="mcc-ports-help-cmd">
                            <code>cloudflared tunnel route dns TUNNEL random-words.YOUR-DOMAIN</code>
                        </div>
                        <span>Uses your own domain with a named Cloudflare tunnel. A random subdomain (e.g. <code>brave-lake-fern.your-domain.xyz</code>) is generated for each port to prevent guessing. Requires <code>cloudflared</code> setup and <code>base_domain</code> in the server config file.</span>
                    </div>
                    <div className="mcc-ports-help-provider">
                        <strong>Cloudflare (My Domain)</strong>
                        <div className="mcc-ports-help-cmd">
                            <code>cloudflared tunnel --hostname random-words.YOUR-DOMAIN</code>
                        </div>
                        <span>Uses your configured owned domain to generate random subdomains. Requires cloudflared authentication and at least one domain configured in Cloudflare Settings.</span>
                    </div>

                    <p className="mcc-ports-help-note">
                        <strong>Note:</strong> localtunnel and Cloudflare Quick tunnels are temporary (URLs change each time, no account needed). Named Cloudflare tunnels use random subdomains on your own domain for security, and require a Cloudflare account with tunnel setup.
                    </p>
                </div>
            )}
        </div>
    );
}
