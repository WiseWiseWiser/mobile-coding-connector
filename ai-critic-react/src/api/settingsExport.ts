// Settings export/import API client and types

import { loadSSHKeys, loadGitHubToken, saveSSHKeys, saveGitHubToken, loadGitUserConfig, saveGitUserConfig, type SSHKey, type GitUserConfig } from '../v2/mcc/home/settings/gitStorage';

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
