import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { exportSettingsZip } from '../../../../api/settingsExport';
import { fetchCredentials } from '../../../../api/auth';
import { fetchEncryptKeyStatus } from '../../../../api/encrypt';
import { fetchDomains } from '../../../../api/domains';
import { fetchCloudflareStatus } from '../../../../api/cloudflare';
import { fetchTerminalConfig } from '../../../../api/terminalConfig';
import { loadSSHKeys, loadGitHubToken, loadGitUserConfig } from './gitStorage';
import './ExportPage.css';

interface ExportStats {
    aiCriticFiles: string;
    cloudflareFiles: string;
    browserData: string;
}

export function ExportPage() {
    const navigate = useNavigate();
    const [includeBrowserData, setIncludeBrowserData] = useState(true);
    const [exporting, setExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [stats, setStats] = useState<Partial<ExportStats>>({});

    useEffect(() => {
        // Load stats for .ai-critic directory contents
        const parts: string[] = [];
        const promises: Promise<void>[] = [];

        promises.push(
            fetchCredentials()
                .then(creds => { if (creds.length > 0) parts.push(`${creds.length} token(s)`); })
                .catch(() => {})
        );
        promises.push(
            fetchEncryptKeyStatus()
                .then(ks => { if (ks.exists) parts.push('encryption keys'); })
                .catch(() => {})
        );
        promises.push(
            fetchDomains()
                .then(resp => { if (resp.domains?.length) parts.push(`${resp.domains.length} domain(s)`); })
                .catch(() => {})
        );
        promises.push(
            fetchTerminalConfig()
                .then(cfg => { if (cfg.extra_paths?.length || cfg.shell) parts.push('terminal config'); })
                .catch(() => {})
        );

        Promise.all(promises).then(() => {
            setStats(prev => ({
                ...prev,
                aiCriticFiles: parts.length > 0 ? parts.join(', ') : 'No files found',
            }));
        });

        // Cloudflare files
        fetchCloudflareStatus()
            .then(s => {
                const count = s.cert_files?.length ?? 0;
                setStats(prev => ({
                    ...prev,
                    cloudflareFiles: count > 0 ? `${count} file(s): ${s.cert_files!.map((f: { name: string }) => f.name).join(', ')}` : 'No files found',
                }));
            })
            .catch(() => setStats(prev => ({ ...prev, cloudflareFiles: 'Unable to load' })));

        // Browser data
        const sshKeys = loadSSHKeys();
        const token = loadGitHubToken();
        const gitUserConfig = loadGitUserConfig();
        const browserParts: string[] = [];
        if (sshKeys.length > 0) browserParts.push(`${sshKeys.length} SSH key(s)`);
        if (token) browserParts.push('GitHub token');
        if (gitUserConfig.name || gitUserConfig.email) {
            browserParts.push(`Git user: ${gitUserConfig.name || '(no name)'}`);
        }
        setStats(prev => ({
            ...prev,
            browserData: browserParts.length > 0 ? browserParts.join(', ') : 'No data',
        }));
    }, []);

    const handleExport = async () => {
        setExporting(true);
        setError(null);
        try {
            await exportSettingsZip(includeBrowserData);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setExporting(false);
    };

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Export Settings</h2>
            </div>

            <div className="export-page-card">
                <p className="export-page-description">
                    Export all configuration as a .zip file. This includes all files under the .ai-critic directory and Cloudflare credentials.
                </p>

                <div className="export-page-sections">
                    <div className="export-page-section-item export-page-section-item--always">
                        <div className="export-page-section-info">
                            <span className="export-page-section-label">.ai-critic/ directory</span>
                            {stats.aiCriticFiles && (
                                <span className="export-page-section-stat">{stats.aiCriticFiles}</span>
                            )}
                            <span className="export-page-section-note">Server credentials, encryption keys, domains, terminal config, etc.</span>
                        </div>
                    </div>

                    <div className="export-page-section-item export-page-section-item--always">
                        <div className="export-page-section-info">
                            <span className="export-page-section-label">Cloudflare credentials</span>
                            {stats.cloudflareFiles && (
                                <span className="export-page-section-stat">{stats.cloudflareFiles}</span>
                            )}
                            <span className="export-page-section-note">Files from ~/.cloudflared/ (cert.pem, tunnel JSONs)</span>
                        </div>
                    </div>

                    <label className="export-page-section-item">
                        <input
                            type="checkbox"
                            checked={includeBrowserData}
                            onChange={() => setIncludeBrowserData(prev => !prev)}
                        />
                        <div className="export-page-section-info">
                            <span className="export-page-section-label">Browser Data (Git Configs)</span>
                            {stats.browserData && (
                                <span className="export-page-section-stat">{stats.browserData}</span>
                            )}
                            <span className="export-page-section-note">SSH keys, GitHub token, git user config from browser storage</span>
                        </div>
                    </label>
                </div>

                {error && <div className="export-page-error">{error}</div>}

                <div className="export-page-actions">
                    <button
                        className="export-page-btn export-page-btn--primary"
                        onClick={handleExport}
                        disabled={exporting}
                    >
                        {exporting ? 'Exporting...' : 'Export .zip'}
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
