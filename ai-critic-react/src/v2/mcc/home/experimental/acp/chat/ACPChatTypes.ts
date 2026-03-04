export interface ChatMessage {
    role: 'user' | 'agent';
    content: string;
    toolCalls?: ToolCallInfo[];
    plan?: PlanEntry[];
}

export interface ToolCallInfo {
    id: string;
    title: string;
    status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
    content?: string;
}

export interface PlanEntry {
    content: string;
    status: 'pending' | 'in_progress' | 'completed';
    priority?: string;
}

const ConnectionStatuses = {
    Disconnected: 'disconnected',
    Connecting: 'connecting',
    Connected: 'connected',
    Error: 'error',
} as const;

export { ConnectionStatuses };
export type ConnectionStatus = typeof ConnectionStatuses[keyof typeof ConnectionStatuses];
