export const CODEX_DEFAULT_MODEL_KEY = 'mcc.codex.default-model.v1';
export const CODEX_DEFAULT_REASONING_EFFORT_KEY = 'mcc.codex.default-reasoning-effort.v1';
export const CODEX_SANDBOX_KEY = 'mcc.codex.sandbox.v1';
export const CODEX_APPROVAL_POLICY_KEY = 'mcc.codex.approval-policy.v1';

export const CODEX_DEFAULT_SANDBOX = 'danger-full-access';
export const CODEX_DEFAULT_APPROVAL_POLICY = 'never';

export interface CodexModel {
    id: string;
    name: string;
    description?: string;
    defaultReasoningLevel?: string;
    reasoningLevels?: string[];
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

export function loadCodexDefaultReasoningEffort(): string {
    return window.localStorage.getItem(CODEX_DEFAULT_REASONING_EFFORT_KEY) || '';
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

export function saveCodexDefaultReasoningEffort(value: string) {
    if (value) {
        window.localStorage.setItem(CODEX_DEFAULT_REASONING_EFFORT_KEY, value);
        return;
    }
    window.localStorage.removeItem(CODEX_DEFAULT_REASONING_EFFORT_KEY);
}

export async function fetchCodexModels(): Promise<{ models: CodexModel[]; currentModel: string; currentReasoningLevel: string }> {
    const response = await fetch('/api/agents/codex/models');
    if (!response.ok) {
        throw new Error(await response.text());
    }
    const data = await response.json();
    return {
        models: Array.isArray(data.models) ? data.models.map((item: any) => ({
            id: typeof item.id === 'string' ? item.id : '',
            name: typeof item.name === 'string' ? item.name : '',
            description: typeof item.description === 'string' ? item.description : undefined,
            defaultReasoningLevel: typeof item.default_reasoning_level === 'string' ? item.default_reasoning_level : undefined,
            reasoningLevels: Array.isArray(item.reasoning_levels) ? item.reasoning_levels.filter((level: unknown): level is string => typeof level === 'string') : undefined,
        })).filter((item: CodexModel) => item.id) : [],
        currentModel: typeof data.current_model === 'string' ? data.current_model : '',
        currentReasoningLevel: typeof data.current_reasoning_level === 'string' ? data.current_reasoning_level : '',
    };
}
