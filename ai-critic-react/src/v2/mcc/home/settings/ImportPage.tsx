import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ALL_SECTIONS, applyImportData, type ExportSectionKey, type SettingsExportData } from '../../../../api/settingsExport';
import { fetchCredentials } from '../../../../api/auth';
import { fetchEncryptKeyStatus } from '../../../../api/encrypt';
import { fetchDomains } from '../../../../api/domains';
import { fetchCloudflareStatus } from '../../../../api/cloudflare';
import { fetchTerminalConfig } from '../../../../api/terminalConfig';
import { loadSSHKeys, loadGitHubToken, loadGitUserConfig } from './gitStorage';
import './ImportPage.css';

type ImportStep = 'choose' | 'preview' | 'done';

interface SystemStats {
    credentials: string;
    encryption_keys: string;
    web_domains: string;
    cloudflare_auth: string;
    git_configs: string;
    terminal_config: string;
}

export function ImportPage() {
    const navigate = useNavigate();
    const fileInputRef = useRef<HTMLInputElement>(null);

    const [step, setStep] = useState<ImportStep>('choose');
    const [data, setData] = useState<SettingsExportData | null>(null);
    const [parseError, setParseError] = useState<string | null>(null);
    const [selected, setSelected] = useState<Set<ExportSectionKey>>(new Set());
    const [importing, setImporting] = useState(false);
    const [importError, setImportError] = useState<string | null>(null);
    const [systemStats, setSystemStats] = useState<Partial<SystemStats>>({});

    // Load existing system stats
    useEffect(() => {
        fetchCredentials()
            .then(creds => setSystemStats(prev => ({ ...prev, credentials: `${creds.length} token(s)` })))
            .catch(() => setSystemStats(prev => ({ ...prev, credentials: 'Unable to load' })));

        fetchEncryptKeyStatus()
            .then(ks => setSystemStats(prev => ({ ...prev, encryption_keys: ks.valid ? '1 key pair (valid)' : ks.exists ? '1 key pair (invalid)' : 'No key pair' })))
            .catch(() => setSystemStats(prev => ({ ...prev, encryption_keys: 'Unable to load' })));

        fetchDomains()
            .then(resp => setSystemStats(prev => ({ ...prev, web_domains: `${resp.domains?.length ?? 0} domain(s)` })))
            .catch(() => setSystemStats(prev => ({ ...prev, web_domains: 'Unable to load' })));

        fetchCloudflareStatus()
            .then(s => {
                const count = s.cert_files?.length ?? 0;
                setSystemStats(prev => ({ ...prev, cloudflare_auth: `${count} auth file(s)` }));
            })
            .catch(() => setSystemStats(prev => ({ ...prev, cloudflare_auth: 'Unable to load' })));

        const sshKeys = loadSSHKeys();
        const token = loadGitHubToken();
        const gitUserConfig = loadGitUserConfig();
        const parts: string[] = [];
        parts.push(`${sshKeys.length} SSH key(s)`);
        if (token) parts.push('GitHub token');
        if (gitUserConfig.name || gitUserConfig.email) {
            parts.push(`Git user configured`);
        }
        setSystemStats(prev => ({ ...prev, git_configs: parts.join(', ') }));

        fetchTerminalConfig()
            .then(cfg => {
                const count = cfg.extra_paths?.length ?? 0;
                setSystemStats(prev => ({ ...prev, terminal_config: `${count} extra PATH(s)` }));
            })
            .catch(() => setSystemStats(prev => ({ ...prev, terminal_config: 'Unable to load' })));
    }, []);

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setParseError(null);
        const reader = new FileReader();
        reader.onload = () => {
            try {
                const parsed = JSON.parse(reader.result as string) as SettingsExportData;
                if (!parsed.version || !parsed.sections) {
                    setParseError('Invalid settings file: missing version or sections.');
                    return;
                }
                setData(parsed);
                const available = new Set<ExportSectionKey>();
                for (const s of ALL_SECTIONS) {
                    if (parsed.sections[s.key]) {
                        available.add(s.key);
                    }
                }
                setSelected(available);
                setStep('preview');
            } catch {
                setParseError('Failed to parse JSON file. Please check the file format.');
            }
        };
        reader.readAsText(file);
    };

    const toggle = (key: ExportSectionKey) => {
        setSelected(prev => {
            const next = new Set(prev);
            if (next.has(key)) {
                next.delete(key);
            } else {
                next.add(key);
            }
            return next;
        });
    };

    const handleConfirm = async () => {
        if (!data || selected.size === 0) return;
        setImporting(true);
        setImportError(null);
        try {
            await applyImportData(data, [...selected]);
            setStep('done');
        } catch (err) {
            setImportError(err instanceof Error ? err.message : String(err));
        }
        setImporting(false);
    };

    const fileSummary = (key: ExportSectionKey): string => {
        if (!data?.sections[key]) return '';
        switch (key) {
            case 'credentials': {
                const cr = data.sections.credentials!;
                return `${cr.tokens.length} token(s)`;
            }
            case 'encryption_keys':
                return 'Private key + public key';
            case 'web_domains': {
                const wd = data.sections.web_domains!;
                return `${wd.domains.length} domain(s)${wd.tunnel_name ? `, tunnel: ${wd.tunnel_name}` : ''}`;
            }
            case 'cloudflare_auth': {
                const ca = data.sections.cloudflare_auth!;
                return `${ca.files.length} file(s): ${ca.files.map(f => f.name).join(', ')}`;
            }
            case 'git_configs': {
                const gc = data.sections.git_configs!;
                const parts: string[] = [];
                if (gc.ssh_keys?.length) parts.push(`${gc.ssh_keys.length} SSH key(s)`);
                if (gc.github_token) parts.push('GitHub token');
                if (gc.git_user_config?.name || gc.git_user_config?.email) {
                    parts.push('Git user config');
                }
                return parts.join(', ') || 'Empty';
            }
            case 'terminal_config': {
                const tc = data.sections.terminal_config!;
                const parts: string[] = [];
                parts.push(`${tc.extra_paths?.length ?? 0} extra PATH(s)`);
                if (tc.shell) parts.push(`shell: ${tc.shell}`);
                if (tc.shell_flags?.length) parts.push(`flags: ${tc.shell_flags.join(' ')}`);
                return parts.join(', ');
            }
            default:
                return '';
        }
    };

    const getExistingStat = (key: ExportSectionKey): string | undefined => systemStats[key as keyof SystemStats];

    const getNote = (key: ExportSectionKey): string | undefined => {
        if (key === 'git_configs') return 'Will be saved to browser local storage only';
        if (key === 'credentials') return 'Tokens will be merged (deduplicated) with existing';
        return undefined;
    };

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Import Settings</h2>
            </div>

            {step === 'choose' && (
                <div className="import-page-card">
                    <p className="import-page-description">
                        Choose a previously exported settings file (.json) to import.
                    </p>

                    <input
                        ref={fileInputRef}
                        type="file"
                        accept=".json,application/json"
                        className="import-page-file-input"
                        onChange={handleFileChange}
                    />

                    <button
                        className="import-page-choose-btn"
                        onClick={() => fileInputRef.current?.click()}
                    >
                        Choose File
                    </button>

                    {parseError && <div className="import-page-error">{parseError}</div>}

                    <div className="import-page-actions">
                        <button
                            className="import-page-btn import-page-btn--secondary"
                            onClick={() => navigate('..')}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            {step === 'preview' && data && (
                <div className="import-page-card">
                    <p className="import-page-description">
                        Exported on: {new Date(data.exported_at).toLocaleString()}
                    </p>
                    <p className="import-page-description">
                        Select the sections you want to import. Unchecked sections will be skipped.
                    </p>

                    <div className="import-page-sections">
                        {ALL_SECTIONS.map(s => {
                            const available = !!data.sections[s.key];
                            const existingStat = getExistingStat(s.key);
                            const note = getNote(s.key);
                            return (
                                <label
                                    key={s.key}
                                    className={`import-page-section-item${!available ? ' import-page-section-item--disabled' : ''}`}
                                >
                                    <input
                                        type="checkbox"
                                        checked={selected.has(s.key)}
                                        onChange={() => toggle(s.key)}
                                        disabled={!available}
                                    />
                                    <div className="import-page-section-info">
                                        <span className="import-page-section-label">{s.label}</span>
                                        {available ? (
                                            <>
                                                <span className="import-page-section-summary">
                                                    From file: {fileSummary(s.key)}
                                                </span>
                                                {existingStat && (
                                                    <span className="import-page-section-existing">
                                                        Current system: {existingStat}
                                                    </span>
                                                )}
                                                {note && (
                                                    <span className="import-page-section-note">{note}</span>
                                                )}
                                            </>
                                        ) : (
                                            <span className="import-page-section-summary import-page-section-summary--missing">Not included in export</span>
                                        )}
                                    </div>
                                </label>
                            );
                        })}
                    </div>

                    {importError && <div className="import-page-error">{importError}</div>}

                    <div className="import-page-actions">
                        <button
                            className="import-page-btn import-page-btn--primary"
                            onClick={handleConfirm}
                            disabled={selected.size === 0 || importing}
                        >
                            {importing ? 'Importing...' : 'Confirm'}
                        </button>
                        <button
                            className="import-page-btn import-page-btn--secondary"
                            onClick={() => navigate('..')}
                            disabled={importing}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            {step === 'done' && (
                <div className="import-page-card">
                    <div className="import-page-success">
                        <span className="import-page-success-icon">✓</span>
                        <span>Settings imported successfully!</span>
                    </div>
                    <div className="import-page-actions">
                        <button
                            className="import-page-btn import-page-btn--primary"
                            onClick={() => navigate('..')}
                        >
                            Back to Settings
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
