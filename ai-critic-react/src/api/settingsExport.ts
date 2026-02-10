// Settings export/import API client and types

import { loadSSHKeys, loadGitHubToken, saveSSHKeys, saveGitHubToken, loadGitUserConfig, saveGitUserConfig, type SSHKey, type GitUserConfig } from '../v2/mcc/home/settings/gitStorage';
import { loadCursorAPIKey, saveCursorAPIKey } from '../v2/mcc/agent/cursorStorage';

/** The top-level export JSON structure */
export interface SettingsExportData {
    version: 1;
    exported_at: string;
    sections: SettingsExportSections;
}

export interface TerminalConfigExport {
    extra_paths: string[];
    shell?: string;
    shell_flags?: string[];
}

/** Individual export sections – each is optional (user may deselect) */
export interface SettingsExportSections {
    credentials?: CredentialsExport;
    encryption_keys?: EncryptionKeysExport;
    web_domains?: WebDomainsExport;
    cloudflare_auth?: CloudflareAuthExport;
    git_configs?: GitConfigsExport;
    terminal_config?: TerminalConfigExport;
}

export interface EncryptionKeysExport {
    private_key: string; // PEM-encoded
    public_key: string;  // OpenSSH authorized_key format
}

export interface WebDomainsExport {
    domains: { domain: string; provider: string }[];
    tunnel_name: string;
}

export interface CloudflareAuthExport {
    files: { name: string; content_base64: string }[];
}

export interface CredentialsExport {
    tokens: string[];
}

export interface GitConfigsExport {
    ssh_keys: SSHKey[];
    github_token: string;
    git_user_config?: GitUserConfig;
    cursor_api_key?: string;
}

/** Available section identifiers */
export type ExportSectionKey = keyof SettingsExportSections;

export const ALL_SECTIONS: { key: ExportSectionKey; label: string }[] = [
    { key: 'credentials', label: 'Server Credentials (Auth Tokens)' },
    { key: 'encryption_keys', label: 'Encryption Key Pair' },
    { key: 'web_domains', label: 'Web Domain Configs' },
    { key: 'cloudflare_auth', label: 'Cloudflare Auth Files' },
    { key: 'git_configs', label: 'Git Configs (SSH Keys & GitHub Token)' },
    { key: 'terminal_config', label: 'Terminal Configuration (Extra PATHs)' },
];

/** Fetch server-side export data for selected sections */
export async function fetchServerExport(sections: ExportSectionKey[]): Promise<Partial<SettingsExportSections>> {
    const params = new URLSearchParams();
    for (const s of sections) {
        params.append('section', s);
    }
    const resp = await fetch(`/api/settings/export?${params.toString()}`);
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to export settings');
    }
    return resp.json();
}

/** Import server-side settings */
export async function importServerSettings(sections: Partial<SettingsExportSections>): Promise<void> {
    const resp = await fetch('/api/settings/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(sections),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to import settings');
    }
}

/** Build a full export file, combining server-side and client-side data */
export async function buildExportData(selectedSections: ExportSectionKey[]): Promise<SettingsExportData> {
    // Determine which sections are server-side vs client-side
    const serverSections = selectedSections.filter(s => s !== 'git_configs');
    const includeGit = selectedSections.includes('git_configs');

    const sections: SettingsExportSections = {};

    // Fetch server-side data
    if (serverSections.length > 0) {
        const serverData = await fetchServerExport(serverSections);
        Object.assign(sections, serverData);
    }

    // Gather client-side data (local storage)
    if (includeGit) {
        sections.git_configs = {
            ssh_keys: loadSSHKeys(),
            github_token: loadGitHubToken(),
            git_user_config: loadGitUserConfig(),
            cursor_api_key: loadCursorAPIKey() || undefined,
        };
    }

    return {
        version: 1,
        exported_at: new Date().toISOString(),
        sections,
    };
}

/** Apply import data: server-side sections go to backend, client-side handled locally */
export async function applyImportData(data: SettingsExportData, selectedSections: ExportSectionKey[]): Promise<void> {
    const serverSections: Partial<SettingsExportSections> = {};
    let importGit = false;

    for (const key of selectedSections) {
        const value = data.sections[key];
        if (!value) continue;

        if (key === 'git_configs') {
            importGit = true;
        } else {
            (serverSections as Record<string, unknown>)[key] = value;
        }
    }

    // Import server-side sections
    if (Object.keys(serverSections).length > 0) {
        await importServerSettings(serverSections);
    }

    // Import client-side: git configs
    if (importGit && data.sections.git_configs) {
        const gc = data.sections.git_configs;
        if (gc.ssh_keys && gc.ssh_keys.length > 0) {
            saveSSHKeys(gc.ssh_keys);
        }
        if (gc.github_token) {
            saveGitHubToken(gc.github_token);
        }
        if (gc.git_user_config) {
            saveGitUserConfig(gc.git_user_config);
        }
        if (gc.cursor_api_key) {
            saveCursorAPIKey(gc.cursor_api_key);
        }
    }
}

/** Trigger a JSON file download in the browser */
export function downloadJSON(data: SettingsExportData, filename: string) {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

// ---- Zip-based export/import ----

/** Browser-side data included in zip export */
export interface BrowserExportData {
    git_configs?: GitConfigsExport;
}

/** Export settings as a zip file download. Optionally includes browser-data. */
export async function exportSettingsZip(includeBrowserData: boolean): Promise<void> {
    let browserData: BrowserExportData | undefined;
    if (includeBrowserData) {
        browserData = {
            git_configs: {
                ssh_keys: loadSSHKeys(),
                github_token: loadGitHubToken(),
                git_user_config: loadGitUserConfig(),
                cursor_api_key: loadCursorAPIKey() || undefined,
            },
        };
    }

    const resp = browserData
        ? await fetch('/api/settings/export-zip', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(browserData),
        })
        : await fetch('/api/settings/export-zip');

    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to export settings');
    }

    const blob = await resp.blob();
    const disposition = resp.headers.get('Content-Disposition');
    let filename = 'ai-critic-settings.zip';
    if (disposition) {
        const match = disposition.match(/filename="?([^"]+)"?/);
        if (match) {
            filename = match[1];
        }
    }

    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

/** Preview what files will be created/overwritten/merged on import */
export interface ImportFilePreview {
    path: string;
    action: 'create' | 'overwrite' | 'merge';
    size: number;
}

export interface ImportZipPreviewResult {
    files: ImportFilePreview[];
}

/** Upload a zip file and get a preview of what will happen */
export async function previewImportZip(file: File): Promise<ImportZipPreviewResult> {
    const formData = new FormData();
    formData.append('file', file);

    const resp = await fetch('/api/settings/import-zip/preview', {
        method: 'POST',
        body: formData,
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to preview zip');
    }
    return resp.json();
}

/** Confirm importing a zip file (apply all changes) */
export async function confirmImportZip(file: File): Promise<void> {
    const formData = new FormData();
    formData.append('file', file);

    const resp = await fetch('/api/settings/import-zip/confirm', {
        method: 'POST',
        body: formData,
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to import zip');
    }
}

/** Confirm importing selected files from a zip */
export async function confirmImportZipWithSelection(file: File, selectedPaths: string[]): Promise<void> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('selected_paths', JSON.stringify(selectedPaths));

    const resp = await fetch('/api/settings/import-zip/confirm', {
        method: 'POST',
        body: formData,
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to import zip');
    }
}

/** Extract browser-data.json from a zip file (for client-side import) */
export async function extractBrowserDataFromZip(file: File): Promise<BrowserExportData> {
    const formData = new FormData();
    formData.append('file', file);

    const resp = await fetch('/api/settings/import-zip/browser-data', {
        method: 'POST',
        body: formData,
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to extract browser data');
    }
    return resp.json();
}

/** Apply browser data from zip import to localStorage */
export function applyBrowserData(data: BrowserExportData): void {
    if (data.git_configs) {
        const gc = data.git_configs;
        if (gc.ssh_keys && gc.ssh_keys.length > 0) {
            saveSSHKeys(gc.ssh_keys);
        }
        if (gc.github_token) {
            saveGitHubToken(gc.github_token);
        }
        if (gc.git_user_config) {
            saveGitUserConfig(gc.git_user_config);
        }
        if (gc.cursor_api_key) {
            saveCursorAPIKey(gc.cursor_api_key);
        }
    }
}
