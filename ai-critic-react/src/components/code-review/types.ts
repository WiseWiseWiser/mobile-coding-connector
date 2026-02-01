// Types for the Code Review feature

export interface DiffFile {
    path: string;
    oldPath: string;
    status: string;
    diff: string;
    isStaged: boolean;
    totalLines: number;
}

export interface GitDiffResult {
    workingTreeDiff: string;
    stagedDiff: string;
    files: DiffFile[];
    error?: string;
}

export interface ProviderInfo {
    name: string;
}

export interface ModelInfo {
    provider: string;
    model: string;
    displayName?: string;
}

export interface ConfigResponse {
    initialDir: string;
    providers?: ProviderInfo[];
    models?: ModelInfo[];
    defaultProvider?: string;
    defaultModel?: string;
}
