export interface ScreenDisplay {
    id: string;
    name: string;
    x: number;
    y: number;
    w: number;
    h: number;
    scale: number;
    primary: boolean;
}

export interface ScreenCaptureInfo {
    tool: string;
    width: number;
    height: number;
    fps: number;
    idle_fps: number;
    tile_size: number;
}

export interface ScreenControlInfo {
    enabled: boolean;
    holder: string;
    since: number;
}

export interface ScreenInputInfo {
    available: boolean;
    reason?: string;
    permission?: string;
    can_prompt: boolean;
    backend: string;
    /** True while the host login window owns the display; input is refused. */
    locked?: boolean;
}

export interface ScreenStatus {
    state: 'stopped' | 'starting' | 'streaming' | 'error';
    error?: string;
    platform: string;
    displays: ScreenDisplay[] | null;
    capture: ScreenCaptureInfo;
    control: ScreenControlInfo;
    input: ScreenInputInfo;
    viewers: number;
}

export interface ScreenConfig {
    enabled: boolean;
    allow_control: boolean;
    max_width: number;
    fps: number;
    idle_fps: number;
    quality: number;
    max_viewers: number;
}

async function json<T>(resp: Response): Promise<T> {
    if (!resp.ok) {
        let detail = '';
        try {
            const body = await resp.json();
            detail = body?.error ? `: ${body.error}` : '';
        } catch {
            // body was not JSON; the status code is all we have
        }
        throw new Error(`request failed (${resp.status})${detail}`);
    }
    return resp.json() as Promise<T>;
}

export async function fetchScreenStatus(): Promise<ScreenStatus> {
    return json<ScreenStatus>(await fetch('/api/screen/status'));
}

export async function fetchScreenConfig(): Promise<ScreenConfig> {
    return json<ScreenConfig>(await fetch('/api/screen/config'));
}

export async function saveScreenConfig(config: Partial<ScreenConfig>): Promise<ScreenConfig> {
    const resp = await fetch('/api/screen/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
    });
    return json<ScreenConfig>(resp);
}

/** Kill switch: drop whoever controls the desktop right now. */
export async function releaseScreenControl(): Promise<void> {
    await json<unknown>(await fetch('/api/screen/control/release', { method: 'POST' }));
}

/** macOS: open the exact Privacy pane and register this binary in the list. */
export async function promptScreenPermission(which: 'accessibility' | 'screen'): Promise<void> {
    await json<unknown>(await fetch('/api/screen/permission/prompt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ which }),
    }));
}
