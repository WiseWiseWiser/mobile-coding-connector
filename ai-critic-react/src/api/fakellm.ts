export interface ProcessRequest {
    prompt: string;
    project_path?: string;
}

export interface ProcessResponse {
    output: string;
    status: string;
    duration_ms: number;
}

export interface StreamEvent {
    type: 'start' | 'step' | 'done' | 'aborted';
    step?: string;
    message?: string;
    progress?: number;
    output?: string;
    timestamp: number;
}

export interface StreamCallbacks {
    onStart?: () => void;
    onStep?: (event: StreamEvent) => void;
    onDone?: (event: StreamEvent) => void;
    onAborted?: (event: StreamEvent) => void;
    onError?: (error: Error) => void;
}

export async function processPrompt(prompt: string, projectPath?: string): Promise<ProcessResponse> {
    const body: ProcessRequest = { prompt };
    if (projectPath) body.project_path = projectPath;

    const res = await fetch('/api/fakellm/process', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to process' }));
        throw new Error(err.error || 'Failed to process');
    }

    return res.json();
}

export async function streamProcess(
    prompt: string,
    projectPath: string | undefined,
    callbacks: StreamCallbacks
): Promise<void> {
    const body: ProcessRequest = { prompt };
    if (projectPath) body.project_path = projectPath;

    const res = await fetch('/api/fakellm/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to start stream' }));
        callbacks.onError?.(new Error(err.error || 'Failed to start stream'));
        return;
    }

    const reader = res.body?.getReader();
    if (!reader) {
        callbacks.onError?.(new Error('Failed to read response stream'));
        return;
    }

    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
            if (!line.startsWith('data: ')) continue;
            try {
                const event: StreamEvent = JSON.parse(line.slice(6));
                switch (event.type) {
                    case 'start':
                        callbacks.onStart?.();
                        break;
                    case 'step':
                        callbacks.onStep?.(event);
                        break;
                    case 'done':
                        callbacks.onDone?.(event);
                        break;
                    case 'aborted':
                        callbacks.onAborted?.(event);
                        break;
                }
            } catch {
                // Skip malformed SSE data
            }
        }
    }
}

export async function stopProcess(): Promise<{ status: string }> {
    const res = await fetch('/api/fakellm/stop', {
        method: 'POST',
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to stop' }));
        throw new Error(err.error || 'Failed to stop');
    }

    return res.json();
}

export async function getStatus(): Promise<{ status: string; sessionID?: string }> {
    const res = await fetch('/api/fakellm/status');
    return res.json();
}
