import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ALL_SECTIONS, buildExportData, downloadJSON, type ExportSectionKey } from '../../../../api/settingsExport';
import { fetchCredentials } from '../../../../api/auth';
import { fetchEncryptKeyStatus } from '../../../../api/encrypt';
import { fetchDomains } from '../../../../api/domains';
import { fetchCloudflareStatus } from '../../../../api/cloudflare';
import { fetchTerminalConfig } from '../../../../api/terminalConfig';
import { loadSSHKeys, loadGitHubToken } from './gitStorage';
import './ExportPage.css';

interface SectionStats {
    credentials: string;
    encryption_keys: string;
    web_domains: string;
    cloudflare_auth: string;
    git_configs: string;
    terminal_config: string;
}

export function ExportPage() {
    const navigate = useNavigate();
    const [selected, setSelected] = useState<Set<ExportSectionKey>>(
        new Set(ALL_SECTIONS.map(s => s.key))
    );
    const [exporting, setExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [stats, setStats] = useState<Partial<SectionStats>>({});

    useEffect(() => {
        // Load stats for each section in parallel
        fetchCredentials()
            .then(creds => setStats(prev => ({ ...prev, credentials: `${creds.length} token(s)` })))
            .catch(() => setStats(prev => ({ ...prev, credentials: 'Unable to load' })));

        fetchEncryptKeyStatus()
            .then(ks => setStats(prev => ({ ...prev, encryption_keys: ks.valid ? '1 key pair (valid)' : ks.exists ? '1 key pair (invalid)' : 'No key pair' })))
            .catch(() => setStats(prev => ({ ...prev, encryption_keys: 'Unable to load' })));

        fetchDomains()
            .then(resp => setStats(prev => ({ ...prev, web_domains: `${resp.domains?.length ?? 0} domain(s)` })))
            .catch(() => setStats(prev => ({ ...prev, web_domains: 'Unable to load' })));

        fetchCloudflareStatus()
            .then(s => {
                const count = s.cert_files?.length ?? 0;
                setStats(prev => ({ ...prev, cloudflare_auth: `${count} auth file(s)` }));
            })
            .catch(() => setStats(prev => ({ ...prev, cloudflare_auth: 'Unable to load' })));

        // Git configs from local storage
        const sshKeys = loadSSHKeys();
        const token = loadGitHubToken();
        const parts: string[] = [];
        parts.push(`${sshKeys.length} SSH key(s)`);
        if (token) parts.push('GitHub token');
        setStats(prev => ({ ...prev, git_configs: parts.join(', ') }));

        fetchTerminalConfig()
            .then(cfg => {
                const parts: string[] = [];
                parts.push(`${cfg.extra_paths?.length ?? 0} extra PATH(s)`);
                if (cfg.shell) parts.push(`shell: ${cfg.shell}`);
                setStats(prev => ({ ...prev, terminal_config: parts.join(', ') }));
            })
            .catch(() => setStats(prev => ({ ...prev, terminal_config: 'Unable to load' })));
    }, []);

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

    const handleExport = async () => {
        if (selected.size === 0) return;
        setExporting(true);
        setError(null);
        try {
            const data = await buildExportData([...selected]);
            const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
            downloadJSON(data, `ai-critic-settings-${timestamp}.json`);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setExporting(false);
    };

    const getStat = (key: ExportSectionKey): string | undefined => stats[key as keyof SectionStats];
    const getNote = (key: ExportSectionKey): string | undefined => {
        if (key === 'git_configs') return 'Read from browser local storage';
        return undefined;
    };

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Export Settings</h2>
            </div>

            <div className="export-page-card">
                <p className="export-page-description">
                    Select the configuration sections to export. The data will be saved as a JSON file.
                </p>

                <div className="export-page-sections">
                    {ALL_SECTIONS.map(s => (
                        <label key={s.key} className="export-page-section-item">
                            <input
                                type="checkbox"
                                checked={selected.has(s.key)}
                                onChange={() => toggle(s.key)}
                            />
                            <div className="export-page-section-info">
                                <span className="export-page-section-label">{s.label}</span>
                                {getStat(s.key) && (
                                    <span className="export-page-section-stat">{getStat(s.key)}</span>
                                )}
                                {getNote(s.key) && (
                                    <span className="export-page-section-note">{getNote(s.key)}</span>
                                )}
                            </div>
                        </label>
                    ))}
                </div>

                {error && <div className="export-page-error">{error}</div>}

                <div className="export-page-actions">
                    <button
                        className="export-page-btn export-page-btn--primary"
                        onClick={handleExport}
                        disabled={selected.size === 0 || exporting}
                    >
                        {exporting ? 'Exporting...' : 'Export'}
                    </button>
                    <button
                        className="export-page-btn export-page-btn--secondary"
                        onClick={() => navigate('..')}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
}
