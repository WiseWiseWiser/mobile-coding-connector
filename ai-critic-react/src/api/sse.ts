// Shared utility for consuming SSE (Server-Sent Events) streams from the backend.

import type { LogLine } from '../v2/LogViewer';

export interface SSEStreamCallbacks {
    onLog: (line: LogLine) => void;
    onError: (line: LogLine) => void;
    onDone: (message: string, data: Record<string, string>) => void;
    /** Optional handler for custom event types not covered by log/error/done. */
    onCustom?: (data: Record<string, string>) => void;
}

/**
 * Reads an SSE stream from a fetch Response and dispatches events via callbacks.
 *
 * The backend sends events in the format:
 *   data: {"type":"log","message":"..."}
 *   data: {"type":"error","message":"..."}
 *   data: {"type":"done","message":"...","key":"value",...}
 *
 * Returns true if the stream completed with a "done" event, false otherwise.
 */
export async function consumeSSEStream(resp: Response, callbacks: SSEStreamCallbacks): Promise<boolean> {
    const reader = resp.body?.getReader();
    if (!reader) {
        callbacks.onError({ text: 'Failed to read response stream', error: true });
        return false;
    }

    const decoder = new TextDecoder();
    let buffer = '';
    let gotDone = false;

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
            if (!line.startsWith('data: ')) continue;
            try {
                const data = JSON.parse(line.slice(6));
                if (data.type === 'log') {
                    callbacks.onLog({ text: data.message });
                } else if (data.type === 'error') {
                    callbacks.onError({ text: data.message, error: true });
                } else if (data.type === 'done') {
                    gotDone = true;
                    callbacks.onDone(data.message, data);
                } else if (callbacks.onCustom) {
                    callbacks.onCustom(data);
                }
            } catch {
                // Skip malformed SSE data
            }
        }
    }

    return gotDone;
}
