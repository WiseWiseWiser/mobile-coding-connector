export const CODEX_DEFAULT_MODEL_KEY = 'mcc.codex.default-model.v1';
export const CODEX_SANDBOX_KEY = 'mcc.codex.sandbox.v1';
export const CODEX_APPROVAL_POLICY_KEY = 'mcc.codex.approval-policy.v1';

export const CODEX_DEFAULT_SANDBOX = 'danger-full-access';
export const CODEX_DEFAULT_APPROVAL_POLICY = 'never';

export interface CodexModel {
    id: string;
    name: string;
    description?: string;
}

export function loadCodexSandbox(): string {
    return window.localStorage.getItem(CODEX_SANDBOX_KEY) || CODEX_DEFAULT_SANDBOX;
}

export function loadCodexApprovalPolicy(): string {
    return window.localStorage.getItem(CODEX_APPROVAL_POLICY_KEY) || CODEX_DEFAULT_APPROVAL_POLICY;
}

export function loadCodexDefaultModel(): string {
    return window.localStorage.getItem(CODEX_DEFAULT_MODEL_KEY) || '';
}

export function saveCodexSandbox(value: string) {
    window.localStorage.setItem(CODEX_SANDBOX_KEY, value);
}

export function saveCodexApprovalPolicy(value: string) {
    window.localStorage.setItem(CODEX_APPROVAL_POLICY_KEY, value);
}

export function saveCodexDefaultModel(value: string) {
    if (value) {
        window.localStorage.setItem(CODEX_DEFAULT_MODEL_KEY, value);
        return;
    }
    window.localStorage.removeItem(CODEX_DEFAULT_MODEL_KEY);
}

export async function fetchCodexModels(): Promise<{ models: CodexModel[]; currentModel: string }> {
    const response = await fetch('/api/agents/codex/models');
    if (!response.ok) {
        throw new Error(await response.text());
    }
    const data = await response.json();
    return {
        models: Array.isArray(data.models) ? data.models : [],
        currentModel: typeof data.current_model === 'string' ? data.current_model : '',
    };
}
