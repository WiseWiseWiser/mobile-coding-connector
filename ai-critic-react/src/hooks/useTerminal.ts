import { useEffect, useRef, useState } from 'react';
import type { MutableRefObject } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import type { TerminalTheme } from '../types/terminal';
import { useCurrent } from './useCurrent';

export type { TerminalTheme };

export interface UseTerminalOptions {
    theme?: TerminalTheme;
    fontSize?: number;
    fontFamily?: string;
    /** Working directory for the terminal session */
    cwd?: string;
    /** Display name for the terminal session */
    name?: string;
    /** Command to run automatically after the terminal connects (only for new sessions) */
    initialCommand?: string;
    /** Existing session ID to reconnect to. If provided, reconnects instead of creating a new session */
    sessionId?: string;
    /** Called when the backend assigns a session ID (for new sessions) */
    onSessionId?: (sessionId: string) => void;
    /** Called when the user presses any key after connection is closed, requesting the tab be closed */
    onCloseRequest?: () => void;
    /** Called when Ctrl mode is consumed by a keystroke */
    onCtrlModeConsumed?: () => void;
}

export interface UseTerminalReturn {
    terminalRef: MutableRefObject<HTMLDivElement | null>;
    connected: boolean;
    sendKey: (key: string) => void;
    focus: () => void;
    reconnect: () => void;
    /** Ref to control Ctrl modifier mode. When true, next key input is converted to a control character. */
    ctrlModeRef: MutableRefObject<boolean>;
}

const defaultTheme: TerminalTheme = {
    background: '#1e1e1e',
    foreground: '#d4d4d4',
    cursor: '#d4d4d4',
    cursorAccent: '#1e1e1e',
    selectionBackground: '#264f78',
    black: '#000000',
    red: '#cd3131',
    green: '#0dbc79',
    yellow: '#e5e510',
    blue: '#2472c8',
    magenta: '#bc3fbc',
    cyan: '#11a8cd',
    white: '#e5e5e5',
    brightBlack: '#666666',
    brightRed: '#f14c4c',
    brightGreen: '#23d18b',
    brightYellow: '#f5f543',
    brightBlue: '#3b8eea',
    brightMagenta: '#d670d6',
    brightCyan: '#29b8db',
    brightWhite: '#ffffff',
};

export function useTerminal(
    _isActive: boolean,
    options: UseTerminalOptions = {}
): UseTerminalReturn {
    const [connected, setConnected] = useState(false);
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const cleanupRef = useRef<(() => void) | null>(null);
    const ctrlModeRef = useRef(false);

    // Store options in a ref so setup reads them imperatively
    const optionsRef = useCurrent(options);

    const setupRef = useRef<(() => void) | null>(null);

    // Imperative setup function — creates xterm + WebSocket, stores cleanup in ref
    setupRef.current = () => {
        // Tear down previous instance if any
        cleanupRef.current?.();
        cleanupRef.current = null;

        if (!terminalRef.current) return;

        let disposed = false;
        let createdSessionId = '';  // Track session ID if newly created
        let hadUserInput = false;   // Track if user actually interacted
        const {
            theme = defaultTheme,
            fontSize = 14,
            fontFamily = 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
            cwd,
            name,
            initialCommand,
            sessionId,
        } = optionsRef.current;

        // ---- xterm setup ----
        const xterm = new XTerm({
            cursorBlink: true,
            fontSize,
            fontFamily,
            theme,
            allowProposedApi: true,
            scrollback: 10000,
        });
        const fitAddon = new FitAddon();
        xterm.loadAddon(fitAddon);
        xterm.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = xterm;
        fitAddonRef.current = fitAddon;

        // ---- WebSocket setup ----
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const params = new URLSearchParams();
        if (sessionId) params.set('session_id', sessionId);
        if (cwd) params.set('cwd', cwd);
        if (name) params.set('name', name);
        const queryString = params.toString();
        const wsUrl = `${protocol}//${window.location.host}/api/terminal${queryString ? '?' + queryString : ''}`;
        const ws = new WebSocket(wsUrl);
        ws.binaryType = 'arraybuffer';
        wsRef.current = ws;

        const isReconnecting = !!sessionId;

        const sendResize = () => {
            if (ws.readyState !== WebSocket.OPEN) return;
            const dimensions = fitAddon.proposeDimensions();
            if (dimensions) {
                ws.send(JSON.stringify({ type: 'resize', cols: dimensions.cols, rows: dimensions.rows }));
            }
        };

        ws.onopen = () => {
            if (disposed) return;
            setConnected(true);
            setTimeout(() => {
                if (disposed) return;
                fitAddon.fit();
                sendResize();
                if (!isReconnecting && initialCommand) {
                    setTimeout(() => {
                        if (disposed) return;
                        ws.send(initialCommand + '\n');
                    }, 300);
                }
            }, 100);
        };

        ws.onmessage = (event) => {
            if (disposed) return;
            if (event.data instanceof ArrayBuffer) {
                xterm.write(new TextDecoder().decode(event.data));
            } else {
                const text = event.data as string;
                try {
                    const msg = JSON.parse(text);
                    if (msg.type === 'session_id' && msg.session_id) {
                        createdSessionId = msg.session_id;
                        optionsRef.current.onSessionId?.(msg.session_id);
                        return;
                    }
                } catch {
                    // Not JSON — write as terminal output
                }
                xterm.write(text);
            }
        };

        ws.onerror = () => {
            if (disposed) return;
            xterm.writeln('\r\n\x1b[31mConnection error\x1b[0m');
            setConnected(false);
        };

        ws.onclose = () => {
            if (disposed) return;
            xterm.writeln('\r\n\x1b[33mConnection closed\x1b[0m');
            xterm.writeln('\x1b[90m[Press any key to close]\x1b[0m');
            setConnected(false);

            // Listen for any key press to request tab close
            const keyDisposable = xterm.onKey(() => {
                keyDisposable.dispose();
                optionsRef.current.onCloseRequest?.();
            });
        };

        xterm.onData((data) => {
            if (ws.readyState !== WebSocket.OPEN) return;
            hadUserInput = true;

            // When Ctrl mode is active, convert next single-char input to control character
            if (ctrlModeRef.current && data.length === 1) {
                ctrlModeRef.current = false;
                optionsRef.current.onCtrlModeConsumed?.();
                const charCode = data.charCodeAt(0);
                if (charCode >= 97 && charCode <= 122) {
                    // a-z → Ctrl+A (0x01) through Ctrl+Z (0x1A)
                    ws.send(String.fromCharCode(charCode - 96));
                    return;
                }
                if (charCode >= 65 && charCode <= 90) {
                    // A-Z → Ctrl+A (0x01) through Ctrl+Z (0x1A)
                    ws.send(String.fromCharCode(charCode - 64));
                    return;
                }
            }
            if (ctrlModeRef.current) {
                ctrlModeRef.current = false;
                optionsRef.current.onCtrlModeConsumed?.();
            }
            ws.send(data);
        });

        // ---- Resize handler ----
        const handleResize = () => {
            fitAddonRef.current?.fit();
            sendResize();
        };
        window.addEventListener('resize', handleResize);

        xterm.focus();

        // Store cleanup
        cleanupRef.current = () => {
            disposed = true;
            window.removeEventListener('resize', handleResize);
            if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
                ws.close();
            }
            // If a session was newly created (not reconnecting) and the user never interacted,
            // delete it to avoid orphaned sessions (handles React StrictMode double-mount cleanup)
            if (createdSessionId && !sessionId && !hadUserInput) {
                fetch(`/api/terminal/sessions?id=${encodeURIComponent(createdSessionId)}`, { method: 'DELETE' }).catch(() => {});
            }
            wsRef.current = null;
            xterm.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    };

    // Run setup once on mount, clean up on unmount
    useEffect(() => {
        setupRef.current?.();
        return () => {
            cleanupRef.current?.();
            cleanupRef.current = null;
        };
    }, []);

    const sendKey = (key: string) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(key);
            xtermRef.current?.focus();
        }
    };

    const focus = () => {
        xtermRef.current?.focus();
    };

    const reconnect = () => {
        setupRef.current?.();
    };

    return { terminalRef, connected, sendKey, focus, reconnect, ctrlModeRef };
}
