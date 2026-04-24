import { useEffect, useMemo, useState } from 'react';
import { fetchOwnedDomains } from '../../api/cloudflare';
import { fetchHomeDir } from '../../api/files';
import { consumeSSEStream } from '../../api/sse';
import { deleteService, fetchServices, restartService, saveService, startService, stopService, type ServiceDefinition, type ServiceStatus } from '../../api/services';
import { streamLogFile } from '../../api/logs';
import { useProjectDir } from '../../hooks/project/useProjectDir';
import type { TunnelProvider, ProviderInfo } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { PlusIcon } from '../../pure-view/icons/PlusIcon';
import { LogViewer } from '../LogViewer';
import { ConfirmModal } from './ConfirmModal';

interface ServicesSectionProps {
    availableProviders: ProviderInfo[];
}

interface ServiceFormState {
    id?: string;
    name: string;
    command: string;
    workingDir: string;
    extraEnvText: string;
    enablePortForward: boolean;
    port: string;
    label: string;
    provider: TunnelProvider;
    baseDomain: string;
    subdomain: string;
}

function createDefaultForm(workingDir = ''): ServiceFormState {
    return {
        name: '',
        command: '',
        workingDir,
        extraEnvText: '',
        enablePortForward: false,
        port: '',
        label: '',
        provider: TunnelProviders.Localtunnel,
        baseDomain: '',
        subdomain: '',
    };
}

function formatTime(value?: string): string {
    if (!value) return '';
    try {
        return new Date(value).toLocaleString();
    } catch {
        return value;
    }
}

function appendLogLine<T>(lines: T[], next: T, max = 300): T[] {
    const merged = [...lines, next];
    if (merged.length <= max) return merged;
    return merged.slice(merged.length - max);
}

function stringifyEnvMap(env?: Record<string, string>): string {
    if (!env || Object.keys(env).length === 0) return '';
    return Object.entries(env)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, value]) => `${key}=${value}`)
        .join('\n');
}

function parseEnvText(value: string): { env?: Record<string, string>; error?: string } {
    const lines = value.split(/\r?\n/);
    const env: Record<string, string> = {};
    for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i].trim();
        if (!line || line.startsWith('#')) continue;
        const idx = line.indexOf('=');
        if (idx <= 0) {
            return { error: `Invalid env on line ${i + 1}. Use KEY=VALUE.` };
        }
        const key = line.slice(0, idx).trim();
        const envValue = line.slice(idx + 1);
        if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
            return { error: `Invalid env name on line ${i + 1}: ${key}` };
        }
        env[key] = envValue;
    }
    if (Object.keys(env).length === 0) {
        return {};
    }
    return { env };
}

function toFormState(service: ServiceStatus): ServiceFormState {
    return {
        id: service.id,
        name: service.name,
        command: service.command,
        workingDir: service.workingDir || '',
        extraEnvText: stringifyEnvMap(service.extraEnv),
        enablePortForward: !!service.portForward,
        port: service.portForward?.port ? String(service.portForward.port) : '',
        label: service.portForward?.label || '',
        provider: (service.portForward?.provider as TunnelProvider) || TunnelProviders.Localtunnel,
        baseDomain: service.portForward?.baseDomain || '',
        subdomain: service.portForward?.subdomain || '',
    };
}

export function ServicesSection({ availableProviders }: ServicesSectionProps) {
    const { projectDir } = useProjectDir();
    const [homeDir, setHomeDir] = useState('');
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [actionError, setActionError] = useState<string | null>(null);
    const [showForm, setShowForm] = useState(false);
    const [saving, setSaving] = useState(false);
    const [form, setForm] = useState<ServiceFormState>(createDefaultForm());
    const [deleteTarget, setDeleteTarget] = useState<ServiceStatus | null>(null);

    const providerButtons = useMemo(
        () => availableProviders.filter((provider) => provider.available),
        [availableProviders],
    );

    const refreshServices = async () => {
        try {
            const data = await fetchServices(projectDir || undefined);
            setServices(data);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        setLoading(true);
        refreshServices();
        const timer = setInterval(refreshServices, 3000);
        return () => clearInterval(timer);
    }, [projectDir]);

    useEffect(() => {
        fetchHomeDir()
            .then((dir) => setHomeDir(dir))
            .catch(() => {});
    }, []);

    const resetForm = () => {
        setForm(createDefaultForm(homeDir));
        setShowForm(false);
    };

    const handleSave = async () => {
        const name = form.name.trim();
        const command = form.command.trim();
        const workingDir = form.workingDir.trim();
        if (!name || !command) {
            setActionError('Name and command are required.');
            return;
        }

        const parsedEnv = parseEnvText(form.extraEnvText);
        if (parsedEnv.error) {
            setActionError(parsedEnv.error);
            return;
        }

        let portForward: ServiceDefinition['portForward'];
        if (form.enablePortForward) {
            const port = Number.parseInt(form.port, 10);
            if (!port || port < 1 || port > 65535) {
                setActionError('A valid service port is required.');
                return;
            }
            portForward = {
                port,
                label: form.label.trim() || undefined,
                provider: form.provider,
                baseDomain: form.baseDomain.trim() || undefined,
                subdomain: form.subdomain.trim() || undefined,
            };
        }

        setSaving(true);
        setActionError(null);
        try {
            await saveService({
                id: form.id,
                name,
                command,
                projectDir: projectDir || undefined,
                workingDir: workingDir || undefined,
                extraEnv: parsedEnv.env,
                portForward,
            });
            await refreshServices();
            resetForm();
        } catch (err) {
            setActionError(err instanceof Error ? err.message : String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleAction = async (fn: () => Promise<void>) => {
        setActionError(null);
        try {
            await fn();
            await refreshServices();
        } catch (err) {
            setActionError(err instanceof Error ? err.message : String(err));
        }
    };

    return (
        <section className="mcc-service-section">
            <div className="mcc-section-header">
                <h2>Service</h2>
            </div>
            <div className="mcc-service-subtitle">
                Define commands the server should keep alive, optionally with managed port forwarding.
            </div>

            {error && <div className="mcc-ports-error">Error: {error}</div>}
            {actionError && <div className="mcc-ports-error">{actionError}</div>}

            <div className="mcc-service-list">
                {services.map((service) => (
                    <ServiceCard
                        key={service.id}
                        service={service}
                        onEdit={() => {
                            setForm(toFormState(service));
                            setShowForm(true);
                        }}
                        onStart={() => handleAction(() => startService(service.id))}
                        onStop={() => handleAction(() => stopService(service.id))}
                        onRestart={() => handleAction(() => restartService(service.id))}
                        onDelete={() => setDeleteTarget(service)}
                    />
                ))}
                {!loading && services.length === 0 && (
                    <div className="mcc-ports-empty">No services configured for this scope.</div>
                )}
            </div>

            <div className="mcc-add-port-section">
                {showForm ? (
                    <div className="mcc-add-port-form">
                        <div className="mcc-add-port-header">
                            <span>{form.id ? 'Edit Service' : 'Add Service'}</span>
                            <button className="mcc-close-btn" onClick={resetForm}>×</button>
                        </div>
                        <div className="mcc-service-form-grid">
                            <div className="mcc-form-field">
                                <label>Name</label>
                                <input
                                    type="text"
                                    placeholder="web"
                                    value={form.name}
                                    onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Command</label>
                                <input
                                    type="text"
                                    placeholder="npm run dev"
                                    value={form.command}
                                    onChange={(e) => setForm((prev) => ({ ...prev, command: e.target.value }))}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Working Dir</label>
                                <input
                                    type="text"
                                    placeholder={homeDir || '/home/user'}
                                    value={form.workingDir}
                                    onChange={(e) => setForm((prev) => ({ ...prev, workingDir: e.target.value }))}
                                />
                            </div>
                            <div className="mcc-form-field">
                                <label>Extra Env</label>
                                <textarea
                                    className="mcc-service-env-input"
                                    placeholder={'NODE_ENV=development\nPORT=3000'}
                                    value={form.extraEnvText}
                                    onChange={(e) => setForm((prev) => ({ ...prev, extraEnvText: e.target.value }))}
                                    rows={5}
                                />
                                <div className="mcc-service-field-hint">One <code>KEY=VALUE</code> entry per line.</div>
                            </div>
                        </div>

                        <button
                            type="button"
                            className={`mcc-service-toggle ${form.enablePortForward ? 'active' : ''}`}
                            onClick={() => setForm((prev) => ({ ...prev, enablePortForward: !prev.enablePortForward }))}
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
                                            onChange={(e) => setForm((prev) => ({ ...prev, port: e.target.value }))}
                                        />
                                    </div>
                                    <div className="mcc-form-field">
                                        <label>Forward Label</label>
                                        <input
                                            type="text"
                                            placeholder="app.example.com"
                                            value={form.label}
                                            onChange={(e) => setForm((prev) => ({ ...prev, label: e.target.value }))}
                                        />
                                    </div>
                                </div>
                                <div className="mcc-form-field mcc-provider-field">
                                    <label>Provider</label>
                                    <div className="mcc-provider-options">
                                        {providerButtons.map((provider) => (
                                            <button
                                                key={provider.id}
                                                className={`mcc-provider-btn ${form.provider === provider.id ? 'active' : ''}`}
                                                onClick={() => setForm((prev) => ({ ...prev, provider: provider.id as TunnelProvider }))}
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
                                            onSelect={(domain) => setForm((prev) => ({ ...prev, baseDomain: domain }))}
                                        />
                                        <div className="mcc-form-field mcc-subdomain-field">
                                            <label>Subdomain</label>
                                            <input
                                                type="text"
                                                placeholder="brave-apex-dawn"
                                                value={form.subdomain}
                                                onChange={(e) => setForm((prev) => ({ ...prev, subdomain: e.target.value }))}
                                                className="mcc-subdomain-input"
                                            />
                                        </div>
                                    </>
                                )}
                            </>
                        )}

                        <button className="mcc-forward-btn" onClick={handleSave} disabled={saving}>
                            {saving ? 'Saving...' : form.id ? 'Update Service' : 'Save Service'}
                        </button>
                    </div>
                ) : (
                    <button
                        className="mcc-add-port-btn"
                        onClick={() => {
                            setForm(createDefaultForm(homeDir));
                            setShowForm(true);
                        }}
                    >
                        <PlusIcon />
                        <span>Add Service</span>
                    </button>
                )}
            </div>

            {deleteTarget && (
                <ConfirmModal
                    title="Delete Service"
                    message="Are you sure you want to delete this service?"
                    info={{
                        Name: deleteTarget.name,
                        Command: deleteTarget.command,
                    }}
                    command={`delete service ${deleteTarget.id}`}
                    confirmLabel="Delete Service"
                    confirmVariant="danger"
                    onConfirm={async () => {
                        await handleAction(() => deleteService(deleteTarget.id));
                        setDeleteTarget(null);
                    }}
                    onClose={() => setDeleteTarget(null)}
                />
            )}
        </section>
    );
}

interface ServiceCardProps {
    service: ServiceStatus;
    onEdit: () => void;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onDelete: () => void;
}

function ServiceCard({ service, onEdit, onStart, onStop, onRestart, onDelete }: ServiceCardProps) {
    const [showLogs, setShowLogs] = useState(false);
    const [logLines, setLogLines] = useState<{ text: string; error?: boolean }[]>([]);
    const [streaming, setStreaming] = useState(false);

    useEffect(() => {
        if (!showLogs || !service.logPath) return;

        const controller = new AbortController();
        setLogLines([]);
        setStreaming(true);

        streamLogFile({ path: service.logPath, lines: 100, signal: controller.signal })
            .then(async (response) => {
                await consumeSSEStream(response, {
                    onLog: (line) => setLogLines((prev) => appendLogLine(prev, line)),
                    onError: (line) => setLogLines((prev) => appendLogLine(prev, line)),
                    onDone: (message) => setLogLines((prev) => appendLogLine(prev, { text: message })),
                });
            })
            .catch((err) => {
                if (controller.signal.aborted) return;
                setLogLines((prev) => appendLogLine(prev, { text: err instanceof Error ? err.message : String(err), error: true }));
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setStreaming(false);
                }
            });

        return () => {
            controller.abort();
            setStreaming(false);
        };
    }, [showLogs, service.logPath]);

    const statusLabel = service.status === 'running' ? 'Running' :
        service.status === 'starting' ? 'Starting' :
            service.status === 'error' ? 'Error' : 'Stopped';

    const canStop = service.pid > 0 || service.desiredRunning;

    return (
        <div className={`mcc-port-card mcc-service-card mcc-service-card--${service.status}`}>
            <div className="mcc-service-card-top">
                <div className="mcc-service-card-title-row">
                    <span className="mcc-port-label">{service.name}</span>
                    <span className={`mcc-service-status-badge mcc-service-status-badge--${service.status}`}>{statusLabel}</span>
                </div>
                <div className="mcc-service-command">{service.command}</div>
            </div>

            <div className="mcc-service-meta">
                <span>PID: {service.pid || 'n/a'}</span>
                <span>Scope: {service.projectDir || 'all projects'}</span>
                <span>Working Dir: {service.workingDir || 'home dir'}</span>
            </div>

            <div className="mcc-service-env-block">
                <div className="mcc-service-env-title">Effective PATH</div>
                <div className="mcc-service-env-value">{service.effectivePath || 'Unavailable'}</div>
                {service.extraEnv && Object.keys(service.extraEnv).length > 0 && (
                    <>
                        <div className="mcc-service-env-title">Extra Env</div>
                        <div className="mcc-service-env-value">{stringifyEnvMap(service.extraEnv)}</div>
                    </>
                )}
            </div>

            {service.portForward ? (
                <div className="mcc-service-forward">
                    <div className="mcc-service-forward-line">
                        <span>Port {service.portForward.port}</span>
                        <span>{service.portForward.provider || 'localtunnel'}</span>
                    </div>
                    {service.portForward.publicUrl ? (
                        <a href={service.portForward.publicUrl} target="_blank" rel="noopener noreferrer" className="mcc-port-url-link">
                            {service.portForward.publicUrl}
                        </a>
                    ) : (
                        <div className="mcc-port-url mcc-port-url-connecting">
                            {service.portForward.error || service.portForward.status || 'Waiting for tunnel'}
                        </div>
                    )}
                </div>
            ) : (
                <div className="mcc-service-forward mcc-service-forward--empty">No port forwarding configured.</div>
            )}

            {(service.lastStartedAt || service.lastExitedAt || service.lastExitError) && (
                <div className="mcc-service-times">
                    {service.lastStartedAt && <div>Started: {formatTime(service.lastStartedAt)}</div>}
                    {service.lastExitedAt && <div>Exited: {formatTime(service.lastExitedAt)}</div>}
                    {service.lastExitError && <div className="mcc-service-error-text">Last error: {service.lastExitError}</div>}
                </div>
            )}

            <div className="mcc-port-actions">
                <button type="button" className="mcc-port-action-btn" onClick={onEdit}>Edit</button>
                <button type="button" className="mcc-port-action-btn" onClick={onStart}>Start</button>
                <button type="button" className="mcc-port-action-btn" onClick={onRestart}>Restart</button>
                <button type="button" className="mcc-port-action-btn" onClick={onStop} disabled={!canStop}>Stop</button>
                <button
                    type="button"
                    className={`mcc-port-action-btn mcc-port-logs-btn ${showLogs ? 'active' : ''}`}
                    onClick={() => setShowLogs((prev) => !prev)}
                >
                    Logs
                </button>
                <button type="button" className="mcc-port-action-btn mcc-port-stop" onClick={onDelete}>Delete</button>
            </div>

            {showLogs && (
                <LogViewer
                    lines={logLines}
                    pending={streaming}
                    pendingMessage="Streaming service logs..."
                    className="mcc-port-logs-margin"
                    maxHeight={220}
                />
            )}
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
