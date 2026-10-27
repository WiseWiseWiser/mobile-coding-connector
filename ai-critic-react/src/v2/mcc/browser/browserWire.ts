/** Browser cast wire types, mirroring server/browser. */

export type BrowserMode = 'cast' | 'shot';

/** VIEW only watches; CTRL forwards taps, scrolls and keys to the page. */
export type Lens = 'view' | 'ctrl';

export type PresetName = 'iphone' | 'ipad' | 'mac';

export type ConnectionState = 'connecting' | 'open' | 'closed';

/** One Chrome tab in the current session's window. */
export interface BrowserTab {
    tab_id: number;
    title?: string;
    url?: string;
    /** Chrome is showing this tab right now. */
    active: boolean;
    /** "session_page" for the /go tab, "user" for everything else. */
    role: string;
}

export interface DaemonStatus {
    running: boolean;
    port: number;
    version?: string;
}

/**
 * One rail row. The server only lists sessions it can actually show: the ones
 * with a live Chrome window, plus sessions somebody asked to reopen.
 */
export interface SessionInfo {
    id: string;
    current: boolean;
    attached: boolean;
    attachable: boolean;
    window_id?: number;
    url?: string;
    preset?: string;
    mode?: string;
    viewport?: { width: number; height: number };
    active_tab_id?: number;
    tabs?: BrowserTab[];
    error?: string;
}

export interface SessionsResponse {
    ok: boolean;
    daemon: DaemonStatus;
    current: string;
    sessions: SessionInfo[];
    read_only?: boolean;
    error?: string;
}

export interface TabOpResponse {
    ok: boolean;
    session_id: string;
    tabs: BrowserTab[];
    active_tab_id?: number;
    status?: BrowserStatus;
    error?: string;
}

export interface BrowserStatus {
    ok: boolean;
    daemon: { running: boolean; port: number; version?: string };
    session: {
      id: string;
      connected: boolean;
      window_id?: number;
      content_tab_id?: number;
      url?: string;
      extension_path?: string;
    };
    capture: { active: boolean; viewers: number; fps: number; frames: number; last_ms: number; last_size: number };
    preset: string;
    mode: BrowserMode;
    viewport: { width: number; height: number; dpr?: number };
    presets: { name: string; width: number; height: number }[];
    /** Tabs of the current session's window; the /go page is included. */
    tabs?: BrowserTab[];
    active_tab_id?: number;
    /** Daemon is older than 1.0.20: streaming works, tab controls do not. */
    read_only?: boolean;
    error?: string;
}

/** Frame header: magic, flags, reserved, seq, ts, jpeg length (all big endian). */
export const FRAME_MAGIC = 0x05;
export const FRAME_HEADER_SIZE = 16;

export interface ClientMessage {
    type: string;
    x?: number;
    y?: number;
    button?: number;
    delta_x?: number;
    delta_y?: number;
    key?: string;
    code?: string;
    vk?: number;
    modifiers?: number;
    text?: string;
    url?: string;
    preset?: string;
    mode?: BrowserMode;
}
