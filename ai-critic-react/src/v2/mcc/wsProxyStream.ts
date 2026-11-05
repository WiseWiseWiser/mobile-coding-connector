import { consumeSSEStream } from '../../api/sse';

export type LogLine = { text: string; error?: boolean };
export const START_STEPS = ['binary', 'xray', 'tunnel', 'publish'];

/** streamStart runs the instance start while streaming progress lines. */
export async function streamStart(
    id: string,
    onLine: (text: string, error?: boolean) => void,
    onStep: (step: string) => void,
): Promise<boolean> {
    const resp = await fetch(`/api/ws-proxy/instances/${encodeURIComponent(id)}/start/stream`, { method: 'POST' });
    if (!resp.ok) {
        onLine(await resp.text(), true);
        return false;
    }
    let ok = false;
    await consumeSSEStream(resp, {
        onLog: (l) => onLine(l.text, l.error),
        onError: (l) => onLine(l.text, true),
        onCustom: (data) => {
            if (data?.type === 'status' && typeof data.status === 'string') onStep(data.status);
        },
        onDone: (msg, data) => {
            ok = data?.status === 'running';
            onLine(msg || `start: ${data?.status ?? 'done'}`, data?.status === 'error');
        },
    });
    return ok;
}

/** streamStop runs the stop with a short streamed tail. */
export async function streamStop(id: string, onLine: (text: string, error?: boolean) => void): Promise<void> {
    const resp = await fetch(`/api/ws-proxy/instances/${encodeURIComponent(id)}/stop/stream`, { method: 'POST' });
    if (!resp.ok) {
        onLine(await resp.text(), true);
        return;
    }
    await consumeSSEStream(resp, {
        onLog: (l) => onLine(l.text, l.error),
        onError: (l) => onLine(l.text, true),
        onDone: (msg, data) => onLine(msg || `stop: ${data?.status ?? 'done'}`, data?.status === 'error'),
    });
}
