/**
 * ACP (Agent Communication Protocol) standard types.
 *
 * These types define the standard message format used across all agent adapters
 * (cursor, opencode, etc.) in the frontend. Each adapter converts its native
 * format to/from these standard types.
 */

// ---- Message Types ----

/** Standard ACP message roles */
export const ACPRoles = {
    User: 'user',
    Agent: 'agent',
} as const;
export type ACPRole = typeof ACPRoles[keyof typeof ACPRoles];

/** Standard ACP content types for message parts */
export const ACPContentTypes = {
    TextPlain: 'text/plain',
    ToolCall: 'tool/call',
    ToolResult: 'tool/result',
    Thinking: 'text/thinking',
} as const;
export type ACPContentType = typeof ACPContentTypes[keyof typeof ACPContentTypes];

/** A single part of an ACP message */
export interface ACPMessagePart {
    id: string;
    content_type: ACPContentType;
    content: string;
    /** For tool calls: tool name */
    name?: string;
    /** For tool calls: input/arguments as JSON string */
    metadata?: Record<string, unknown>;
}

/** An ACP message (user or agent) */
export interface ACPMessage {
    id: string;
    role: ACPRole;
    parts: ACPMessagePart[];
    /** Unix timestamp in seconds */
    time?: number;
    /** Model ID used for this message (agent messages only) */
    model?: string;
}

// ---- SSE Event Types ----

/** Standard ACP SSE event types */
export const ACPEventTypes = {
    MessageCreated: 'acp.message.created',
    MessageUpdated: 'acp.message.updated',
    MessageCompleted: 'acp.message.completed',
} as const;
export type ACPEventType = typeof ACPEventTypes[keyof typeof ACPEventTypes];

/** An ACP SSE event payload */
export interface ACPEvent {
    type: ACPEventType;
    message: ACPMessage;
}

// ---- Helper to parse SSE data into ACPEvent ----

/**
 * Parse raw SSE data string into an ACPEvent.
 * Returns null if the data is not a valid ACP event.
 */
export function parseACPEvent(data: string): ACPEvent | null {
    try {
        const parsed = JSON.parse(data);
        if (!parsed.type || !parsed.message) return null;
        return parsed as ACPEvent;
    } catch {
        return null;
    }
}
