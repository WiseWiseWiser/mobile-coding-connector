import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { exportSettingsZip } from '../../../../api/settingsExport';
import { fetchCredentials } from '../../../../api/auth';
import { fetchEncryptKeyStatus } from '../../../../api/encrypt';
import { fetchDomains } from '../../../../api/domains';
import { fetchCloudflareStatus } from '../../../../api/cloudflare';
import { fetchTerminalConfig } from '../../../../api/terminalConfig';
import { fetchOpencodeAuthStatus, type OpencodeAuthStatus } from '../../../../api/agents';
import { loadSSHKeys, loadGitHubToken, loadGitUserConfig } from './gitStorage';
import './ExportPage.css';

interface ExportItem {
    id: string;
    category: string;
    label: string;
    description: string;
    checked: boolean;
    exists: boolean;
    details?: string;
}

export function ExportPage() {
    const navigate = useNavigate();
    const [items, setItems] = useState<ExportItem[]>([]);
    const [exporting, setExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        loadExportItems();
    }, []);

    const loadExportItems = async () => {
        setLoading(true);
        const newItems: ExportItem[] = [];

        // Server Credentials
        try {
            const creds = await fetchCredentials();
            newItems.push({
                id: 'credentials',
                category: 'ai-critic',
                label: 'Server Credentials',
                description: 'API tokens for authentication',
                checked: creds.length > 0,
                exists: creds.length > 0,
                details: creds.length > 0 ? `${creds.length} token(s)` : 'None',
            });
        } catch {
            newItems.push({
                id: 'credentials',
                category: 'ai-critic',
                label: 'Server Credentials',
                description: 'API tokens for authentication',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Encryption Keys
        try {
            const ks = await fetchEncryptKeyStatus();
            newItems.push({
                id: 'encryption_keys',
                category: 'ai-critic',
                label: 'Encryption Keys',
                description: 'RSA key pair for encryption',
                checked: ks.exists,
                exists: ks.exists,
                details: ks.exists ? 'Keys exist' : 'None',
            });
        } catch {
            newItems.push({
                id: 'encryption_keys',
                category: 'ai-critic',
                label: 'Encryption Keys',
                description: 'RSA key pair for encryption',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Web Domains
        try {
            const resp = await fetchDomains();
            const count = resp.domains?.length ?? 0;
            newItems.push({
                id: 'web_domains',
                category: 'ai-critic',
                label: 'Web Domains',
                description: 'Configured domains for port forwarding',
                checked: count > 0,
                exists: count > 0,
                details: count > 0 ? `${count} domain(s)` : 'None',
            });
        } catch {
            newItems.push({
                id: 'web_domains',
                category: 'ai-critic',
                label: 'Web Domains',
                description: 'Configured domains for port forwarding',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Terminal Config
        try {
            const cfg = await fetchTerminalConfig();
            const hasConfig = !!(cfg.extra_paths?.length || cfg.shell);
            newItems.push({
                id: 'terminal_config',
                category: 'ai-critic',
                label: 'Terminal Config',
                description: 'Shell and PATH settings',
                checked: hasConfig,
                exists: hasConfig,
                details: hasConfig ? 'Config exists' : 'None',
            });
        } catch {
            newItems.push({
                id: 'terminal_config',
                category: 'ai-critic',
                label: 'Terminal Config',
                description: 'Shell and PATH settings',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Cloudflare
        try {
            const status = await fetchCloudflareStatus();
            const files = status.cert_files ?? [];
            const hasFiles = files.length > 0;
            newItems.push({
                id: 'cloudflare',
                category: 'cloudflare',
                label: 'Cloudflare Credentials',
                description: 'cert.pem and tunnel files from ~/.cloudflared/',
                checked: hasFiles,
                exists: hasFiles,
                details: hasFiles ? `${files.length} file(s): ${files.map((f: { name: string }) => f.name).join(', ')}` : 'None',
            });
        } catch {
            newItems.push({
                id: 'cloudflare',
                category: 'cloudflare',
                label: 'Cloudflare Credentials',
                description: 'cert.pem and tunnel files from ~/.cloudflared/',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Opencode
        try {
            const status: OpencodeAuthStatus = await fetchOpencodeAuthStatus();
            const authenticated = status.authenticated;
            newItems.push({
                id: 'opencode',
                category: 'opencode',
                label: 'OpenCode Config',
                description: 'Auth, settings & plugins from ~/.local/share/opencode/ and ~/.config/opencode/plugins/',
                checked: authenticated,
                exists: authenticated,
                details: authenticated ? `${status.providers?.length ?? 0} provider(s)` : 'None',
            });
        } catch {
            newItems.push({
                id: 'opencode',
                category: 'opencode',
                label: 'OpenCode Config',
                description: 'Auth, settings & plugins from ~/.local/share/opencode/ and ~/.config/opencode/plugins/',
                checked: false,
                exists: false,
                details: 'Unable to load',
            });
        }

        // Browser Data - SSH Keys
        const sshKeys = loadSSHKeys();
        newItems.push({
            id: 'browser_ssh_keys',
            category: 'browser',
            label: 'SSH Keys',
            description: 'SSH keys stored in browser',
            checked: sshKeys.length > 0,
            exists: sshKeys.length > 0,
            details: sshKeys.length > 0 ? `${sshKeys.length} key(s)` : 'None',
        });

        // Browser Data - GitHub Token
        const token = loadGitHubToken();
        newItems.push({
            id: 'browser_github_token',
            category: 'browser',
            label: 'GitHub Token',
            description: 'GitHub personal access token',
            checked: !!token,
            exists: !!token,
            details: token ? 'Token exists' : 'None',
        });

        // Browser Data - Git Config
        const gitUserConfig = loadGitUserConfig();
        const hasGitConfig = !!(gitUserConfig.name || gitUserConfig.email);
        newItems.push({
            id: 'browser_git_config',
            category: 'browser',
            label: 'Git User Config',
            description: 'Git name and email settings',
            checked: hasGitConfig,
            exists: hasGitConfig,
            details: hasGitConfig ? `${gitUserConfig.name || '(no name)'} <${gitUserConfig.email || '(no email)'}>` : 'None',
        });

        setItems(newItems);
        setLoading(false);
    };

    const toggleItem = (id: string) => {
        setItems(prev => prev.map(item =>
            item.id === id ? { ...item, checked: !item.checked } : item
        ));
    };

    const toggleCategory = (category: string) => {
        const categoryItems = items.filter(i => i.category === category);
        const allChecked = categoryItems.every(i => i.checked);
        setItems(prev => prev.map(item =>
            item.category === category ? { ...item, checked: !allChecked } : item
        ));
    };

    const handleExport = async () => {
        setExporting(true);
        setError(null);
        try {
            const selectedCategories = new Set<string>();
            items.forEach(item => {
                if (item.checked) {
                    selectedCategories.add(item.category);
                }
            });

            await exportSettingsZip({
                includeBrowser: selectedCategories.has('browser'),
                includeAICritic: selectedCategories.has('ai-critic'),
                includeCloudflare: selectedCategories.has('cloudflare'),
                includeOpencode: selectedCategories.has('opencode'),
            });
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setExporting(false);
    };

    const aiCriticItems = items.filter(i => i.category === 'ai-critic');
    const cloudflareItems = items.filter(i => i.category === 'cloudflare');
    const opencodeItems = items.filter(i => i.category === 'opencode');
    const browserItems = items.filter(i => i.category === 'browser');

    const renderCategory = (title: string, items: ExportItem[], categoryId: string) => {
        if (items.length === 0) return null;
        const allChecked = items.every(i => i.checked);
        const someChecked = items.some(i => i.checked) && !allChecked;

        return (
            <div className="export-page-category">
                <div className="export-page-category-header">
                    <label className="export-page-category-checkbox">
                        <input
                            type="checkbox"
                            checked={allChecked}
                            ref={el => {
                                if (el) el.indeterminate = someChecked;
                            }}
                            onChange={() => toggleCategory(categoryId)}
                        />
                        <span className="export-page-category-title">{title}</span>
                    </label>
                </div>
                <div className="export-page-category-items">
                    {items.map(item => (
                        <label key={item.id} className={`export-page-item ${!item.exists ? 'export-page-item--missing' : ''}`}>
                            <input
                                type="checkbox"
                                checked={item.checked}
                                onChange={() => toggleItem(item.id)}
                                disabled={!item.exists}
                            />
                            <div className="export-page-item-info">
                                <span className="export-page-item-label">{item.label}</span>
                                <span className="export-page-item-description">{item.description}</span>
                                {item.details && (
                                    <span className="export-page-item-details">{item.details}</span>
                                )}
                            </div>
                        </label>
                    ))}
                </div>
            </div>
        );
    };

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Export Settings</h2>
            </div>

            <div className="export-page-card">
                <p className="export-page-description">
                    Select the items you want to export. Each category can be expanded to choose specific items.
                </p>

                {loading ? (
                    <div className="export-page-loading">Loading items...</div>
                ) : (
                    <div className="export-page-categories">
                        {renderCategory('.ai-critic/ Directory', aiCriticItems, 'ai-critic')}
                        {renderCategory('Cloudflare Credentials', cloudflareItems, 'cloudflare')}
                        {renderCategory('OpenCode Config', opencodeItems, 'opencode')}
                        {renderCategory('Browser Data (Git Configs)', browserItems, 'browser')}
                    </div>
                )}

                {error && <div className="export-page-error">{error}</div>}

                <div className="export-page-actions">
                    <button
                        className="export-page-btn export-page-btn--primary"
                        onClick={handleExport}
                        disabled={exporting || loading || !items.some(i => i.checked)}
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
