/** Browser cast wire types, mirroring server/browser. */

export type BrowserMode = 'cast' | 'shot';

/** VIEW only watches; CTRL forwards taps, scrolls and keys to the page. */
export type Lens = 'view' | 'ctrl';

export type PresetName = 'iphone' | 'ipad' | 'mac';

export type ConnectionState = 'connecting' | 'open' | 'closed';

/** One Chrome tab in the current session's window. */
export interface BrowserTab {
    tab_id: number;
    /** Chrome's left-to-right tab position; the first page titles a window. */
    index?: number;
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
    /** The title of the window's first content page, when one can be named. */
    title?: string;
    title_tab_id?: number;
    /** A normal Chrome window with no live agent session yet. */
    sessionless?: boolean;
    /** A sessionless window can be adopted by opening a /go page in it. */
    adoptable?: boolean;
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
    /** The extension can list every normal window, not just session windows. */
    windows_supported?: boolean;
    /** The extension can remove a whole Chrome window (window-close). */
    window_close_supported?: boolean;
    read_only?: boolean;
    error?: string;
}

/** POST /api/browser/window/attach adopts one discovered Chrome window. */
export interface AttachWindowResponse {
    ok: boolean;
    session_id: string;
    window_id: number;
    status?: BrowserStatus;
    error?: string;
}

/** POST /api/browser/window/close removes one whole Chrome window. */
export interface CloseWindowResponse {
    ok: boolean;
    closed_window_id?: number;
    closed_session_id?: string;
    current?: string;
    error?: string;
}

export interface TabOpResponse {
    ok: boolean;
    session_id: string;
    tabs: BrowserTab[];
    active_tab_id?: number;
    close_confirmation?: { operation_id: string; tab_id: number };
    close_outcome?: 'closed' | 'kept';
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
    startup?: 'starting' | 'waiting_extension' | 'ready' | 'failed';
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
