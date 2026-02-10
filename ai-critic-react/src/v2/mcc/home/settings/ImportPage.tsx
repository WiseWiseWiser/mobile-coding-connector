import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    previewImportZip,
    confirmImportZipWithSelection,
    extractBrowserDataFromZip,
    applyBrowserData,
    type ImportFilePreview,
    type BrowserExportData,
} from '../../../../api/settingsExport';
import { loadSSHKeys, loadGitHubToken, loadGitUserConfig } from './gitStorage';
import './ImportPage.css';

type ImportStep = 'choose' | 'preview' | 'done';

const FileActions = {
    Create: 'create',
    Overwrite: 'overwrite',
    Merge: 'merge',
} as const;

type FileAction = typeof FileActions[keyof typeof FileActions];

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function getActionLabel(action: FileAction): string {
    switch (action) {
        case FileActions.Create: return 'New';
        case FileActions.Overwrite: return 'Overwrite';
        case FileActions.Merge: return 'Merge';
        default: return action;
    }
}

function getActionClass(action: FileAction): string {
    switch (action) {
        case FileActions.Create: return 'import-page-action-badge--create';
        case FileActions.Overwrite: return 'import-page-action-badge--overwrite';
        case FileActions.Merge: return 'import-page-action-badge--merge';
        default: return '';
    }
}

interface SelectedFile extends ImportFilePreview {
    selected: boolean;
}

/** A single browser data item with its import action */
interface BrowserDataItem {
    label: string;
    detail: string;
    action: FileAction;
}

/** Compare browser data from zip with current localStorage to determine actions */
function computeBrowserDataPreview(data: BrowserExportData): BrowserDataItem[] {
    const items: BrowserDataItem[] = [];

    if (data.git_configs?.ssh_keys?.length) {
        const existing = loadSSHKeys();
        const existingNames = new Set(existing.map(k => k.name));
        const incoming = data.git_configs.ssh_keys;
        const newCount = incoming.filter(k => !existingNames.has(k.name)).length;

        if (existing.length === 0) {
            items.push({
                label: 'SSH Keys',
                detail: `${incoming.length} key(s)`,
                action: FileActions.Create,
            });
        } else if (newCount > 0) {
            items.push({
                label: 'SSH Keys',
                detail: `${incoming.length} key(s) (${newCount} new, ${incoming.length - newCount} existing)`,
                action: FileActions.Merge,
            });
        } else {
            items.push({
                label: 'SSH Keys',
                detail: `${incoming.length} key(s) (all exist)`,
                action: FileActions.Overwrite,
            });
        }
    }

    if (data.git_configs?.github_token) {
        const existing = loadGitHubToken();
        items.push({
            label: 'GitHub Token',
            detail: existing ? 'Will replace existing token' : 'No existing token',
            action: existing ? FileActions.Overwrite : FileActions.Create,
        });
    }

    if (data.git_configs?.git_user_config) {
        const existing = loadGitUserConfig();
        const hasExisting = !!(existing.name || existing.email);
        const gc = data.git_configs.git_user_config;
        items.push({
            label: 'Git User Config',
            detail: `${gc.name || '(no name)'} <${gc.email || '(no email)'}>`,
            action: hasExisting ? FileActions.Overwrite : FileActions.Create,
        });
    }

    return items;
}

/** Group files by their top-level directory for display */
function groupFiles(files: SelectedFile[]): { group: string; files: SelectedFile[] }[] {
    const groups = new Map<string, SelectedFile[]>();

    for (const f of files) {
        const slashIdx = f.path.indexOf('/');
        const group = slashIdx >= 0 ? f.path.substring(0, slashIdx) : '(root)';
        const existing = groups.get(group) || [];
        existing.push(f);
        groups.set(group, existing);
    }

    return Array.from(groups.entries()).map(([group, files]) => ({ group, files }));
}

function getGroupTitle(group: string): string {
    switch (group) {
        case 'ai-critic': return '.ai-critic/ Directory';
        case 'cloudflare': return 'Cloudflare Credentials';
        case 'opencode': return 'OpenCode Config';
        default: return group;
    }
}

export function ImportPage() {
    const navigate = useNavigate();
    const fileInputRef = useRef<HTMLInputElement>(null);

    const [step, setStep] = useState<ImportStep>('choose');
    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [files, setFiles] = useState<SelectedFile[]>([]);
    const [hasBrowserData, setHasBrowserData] = useState(false);
    const [browserDataItems, setBrowserDataItems] = useState<BrowserDataItem[]>([]);
    const [importBrowserData, setImportBrowserData] = useState(true);
    const [parseError, setParseError] = useState<string | null>(null);
    const [importing, setImporting] = useState(false);
    const [importError, setImportError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setParseError(null);
        setLoading(true);

        try {
            // Get preview from server
            const result = await previewImportZip(file);
            setSelectedFile(file);

            // Separate browser-data.json from server files
            const serverFiles = result.files.filter(f => f.path !== 'browser-data.json');
            const browserDataFile = result.files.find(f => f.path === 'browser-data.json');

            // Initialize all files as selected
            setFiles(serverFiles.map(f => ({ ...f, selected: true })));
            setHasBrowserData(!!browserDataFile);
            setImportBrowserData(!!browserDataFile);

            // If browser data exists, fetch it and compute preview
            if (browserDataFile) {
                try {
                    const browserData = await extractBrowserDataFromZip(file);
                    setBrowserDataItems(computeBrowserDataPreview(browserData));
                } catch {
                    setBrowserDataItems([]);
                }
            } else {
                setBrowserDataItems([]);
            }

            setStep('preview');
        } catch (err) {
            setParseError(err instanceof Error ? err.message : String(err));
        }
        setLoading(false);
    };

    const toggleFile = (path: string) => {
        setFiles(prev => prev.map(f =>
            f.path === path ? { ...f, selected: !f.selected } : f
        ));
    };

    const toggleGroup = (group: string) => {
        const groupFiles = files.filter(f => {
            const slashIdx = f.path.indexOf('/');
            const fileGroup = slashIdx >= 0 ? f.path.substring(0, slashIdx) : '(root)';
            return fileGroup === group;
        });
        const allSelected = groupFiles.every(f => f.selected);
        setFiles(prev => prev.map(f => {
            const slashIdx = f.path.indexOf('/');
            const fileGroup = slashIdx >= 0 ? f.path.substring(0, slashIdx) : '(root)';
            return fileGroup === group ? { ...f, selected: !allSelected } : f;
        }));
    };

    const handleConfirm = async () => {
        if (!selectedFile) return;
        setImporting(true);
        setImportError(null);
        try {
            // Get list of selected file paths
            const selectedPaths = files.filter(f => f.selected).map(f => f.path);

            // Import server-side files
            await confirmImportZipWithSelection(selectedFile, selectedPaths);

            // Import browser data if selected
            if (importBrowserData && hasBrowserData) {
                const browserData: BrowserExportData = await extractBrowserDataFromZip(selectedFile);
                applyBrowserData(browserData);
            }

            setStep('done');
        } catch (err) {
            setImportError(err instanceof Error ? err.message : String(err));
        }
        setImporting(false);
    };

    const grouped = groupFiles(files);

    const selectedCount = files.filter(f => f.selected).length;
    const totalCount = files.length;

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Import Settings</h2>
            </div>

            {step === 'choose' && (
                <div className="import-page-card">
                    <p className="import-page-description">
                        Choose a previously exported settings file (.zip) to import.
                    </p>

                    <input
                        ref={fileInputRef}
                        type="file"
                        accept=".zip,application/zip"
                        className="import-page-file-input"
                        onChange={handleFileChange}
                    />

                    <button
                        className="import-page-choose-btn"
                        onClick={() => fileInputRef.current?.click()}
                        disabled={loading}
                    >
                        {loading ? 'Analyzing...' : 'Choose .zip File'}
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

            {step === 'preview' && (
                <div className="import-page-card">
                    <p className="import-page-description">
                        Select the items you want to import from <strong>{selectedFile?.name}</strong>:
                    </p>

                    <div className="import-page-summary">
                        <span className="import-page-summary-badge">
                            {selectedCount} of {totalCount} selected
                        </span>
                    </div>

                    <div className="import-page-file-list">
                        {grouped.map(g => {
                            const allSelected = g.files.every(f => f.selected);
                            const someSelected = g.files.some(f => f.selected) && !allSelected;
                            return (
                                <div key={g.group} className="import-page-file-group">
                                    <div className="import-page-file-group-header">
                                        <label className="import-page-group-checkbox">
                                            <input
                                                type="checkbox"
                                                checked={allSelected}
                                                ref={el => {
                                                    if (el) el.indeterminate = someSelected;
                                                }}
                                                onChange={() => toggleGroup(g.group)}
                                            />
                                            <span>{getGroupTitle(g.group)}/</span>
                                        </label>
                                    </div>
                                    {g.files.map(f => (
                                        <label key={f.path} className="import-page-file-row import-page-file-row--selectable">
                                            <input
                                                type="checkbox"
                                                checked={f.selected}
                                                onChange={() => toggleFile(f.path)}
                                            />
                                            <span className="import-page-file-name">
                                                {f.path.substring(f.path.indexOf('/') + 1)}
                                            </span>
                                            <span className="import-page-file-size">{formatFileSize(f.size)}</span>
                                            <span className={`import-page-action-badge ${getActionClass(f.action as FileAction)}`}>
                                                {getActionLabel(f.action as FileAction)}
                                            </span>
                                        </label>
                                    ))}
                                </div>
                            );
                        })}
                    </div>

                    {hasBrowserData && (
                        <div className="import-page-browser-data-section">
                            <label className="import-page-browser-data-option">
                                <input
                                    type="checkbox"
                                    checked={importBrowserData}
                                    onChange={() => setImportBrowserData(prev => !prev)}
                                />
                                <div className="import-page-section-info">
                                    <span className="import-page-section-label">Browser Data (Git Configs)</span>
                                    <span className="import-page-section-note">
                                        Saved to browser local storage
                                    </span>
                                </div>
                            </label>
                            {importBrowserData && browserDataItems.length > 0 && (
                                <div className="import-page-browser-data-items">
                                    {browserDataItems.map(item => (
                                        <div key={item.label} className="import-page-file-row">
                                            <span className="import-page-file-name">{item.label}</span>
                                            <span className="import-page-file-size">{item.detail}</span>
                                            <span className={`import-page-action-badge ${getActionClass(item.action)}`}>
                                                {getActionLabel(item.action)}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    )}

                    {importError && <div className="import-page-error">{importError}</div>}

                    <div className="import-page-actions">
                        <button
                            className="import-page-btn import-page-btn--primary"
                            onClick={handleConfirm}
                            disabled={selectedCount === 0 || importing}
                        >
                            {importing ? 'Importing...' : `Import ${selectedCount} Item${selectedCount !== 1 ? 's' : ''}`}
                        </button>
                        <button
                            className="import-page-btn import-page-btn--secondary"
                            onClick={() => {
                                setStep('choose');
                                setSelectedFile(null);
                                setFiles([]);
                            }}
                            disabled={importing}
                        >
                            Back
                        </button>
                    </div>
                </div>
            )}

            {step === 'done' && (
                <div className="import-page-card">
                    <div className="import-page-success">
                        <span className="import-page-success-icon">&#10003;</span>
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
