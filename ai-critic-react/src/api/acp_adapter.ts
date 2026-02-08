/**
 * ACP adapter: converts ACP standard events to state updates.
 *
 * The frontend ONLY consumes ACP format. Backend adapters (cursor, opencode)
 * are responsible for converting their native formats to ACP before sending.
 */

import type { ACPMessage } from './acp';
import { ACPEventTypes } from './acp';

// ---- SSE Event Parsing ----

/**
 * Parse an SSE data string (ACP event) and apply it to the messages state.
 * Returns a function that takes the previous messages and returns updated messages,
 * suitable for React's setState(prev => ...) pattern.
 */
export function parseSSEEvent(data: string): ((prev: ACPMessage[]) => ACPMessage[]) | null {
    try {
        const parsed = JSON.parse(data);
        if (!parsed.type || !parsed.message) return null;

        const eventType: string = parsed.type;
        const msg: ACPMessage = parsed.message;

        switch (eventType) {
        case ACPEventTypes.MessageCreated:
            return (prev) => {
                const idx = prev.findIndex(m => m.id === msg.id);
                if (idx >= 0) {
                    const updated = [...prev];
                    updated[idx] = mergeMessage(updated[idx], msg);
                    return updated;
                }
                return [...prev, msg];
            };

        case ACPEventTypes.MessageUpdated:
            return (prev) => {
                const idx = prev.findIndex(m => m.id === msg.id);
                if (idx >= 0) {
                    const updated = [...prev];
                    updated[idx] = mergeMessage(updated[idx], msg);
                    return updated;
                }
                return [...prev, msg];
            };

        case ACPEventTypes.MessageCompleted:
            return (prev) => {
                const idx = prev.findIndex(m => m.id === msg.id);
                if (idx >= 0) {
                    const updated = [...prev];
                    updated[idx] = mergeMessage(updated[idx], msg);
                    return updated;
                }
                return [...prev, msg];
            };

        default:
            return null;
        }
    } catch {
        return null;
    }
}

/**
 * Merge incoming message into existing message.
 * Parts are merged by ID: existing parts are updated, new parts are appended.
 */
function mergeMessage(existing: ACPMessage, incoming: ACPMessage): ACPMessage {
    const mergedParts = [...existing.parts];
    for (const incomingPart of incoming.parts) {
        const idx = mergedParts.findIndex(p => p.id === incomingPart.id);
        if (idx >= 0) {
            mergedParts[idx] = incomingPart;
        } else {
            mergedParts.push(incomingPart);
        }
    }
    return {
        ...existing,
        ...incoming,
        parts: mergedParts,
    };
}

// ---- Message Conversion for REST API responses ----

/**
 * Convert raw messages from the backend to ACPMessage format.
 * The backend should already return ACP format, but this handles any edge cases.
 */
export function convertMessages(raw: unknown[]): ACPMessage[] {
    if (!Array.isArray(raw) || raw.length === 0) return [];
    return raw as ACPMessage[];
}
