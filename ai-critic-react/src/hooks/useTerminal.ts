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
}

export interface UseTerminalReturn {
    terminalRef: MutableRefObject<HTMLDivElement | null>;
    connected: boolean;
    sendKey: (key: string) => void;
    focus: () => void;
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
    isActive: boolean,
    options: UseTerminalOptions = {}
): UseTerminalReturn {
    const [connected, setConnected] = useState(false);
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const wsRef = useRef<WebSocket | null>(null);

    const {
        theme = defaultTheme,
        fontSize = 14,
        fontFamily = 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
        cwd,
        name,
        initialCommand,
        sessionId,
    } = options;

    const onSessionIdRef = useCurrent(options.onSessionId);

    useEffect(() => {
        if (!isActive || !terminalRef.current) {
            return;
        }

        // Initialize xterm
        const xterm = new XTerm({
            cursorBlink: true,
            fontSize,
            fontFamily,
            theme,
        });

        const fitAddon = new FitAddon();
        xterm.loadAddon(fitAddon);

        xterm.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = xterm;
        fitAddonRef.current = fitAddon;

        // Build WebSocket URL with query params
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const params = new URLSearchParams();
        if (sessionId) {
            params.set('session_id', sessionId);
        }
        if (cwd) {
            params.set('cwd', cwd);
        }
        if (name) {
            params.set('name', name);
        }
        const queryString = params.toString();
        const wsUrl = `${protocol}//${window.location.host}/api/terminal${queryString ? '?' + queryString : ''}`;
        const ws = new WebSocket(wsUrl);
        ws.binaryType = 'arraybuffer';
        wsRef.current = ws;

        const isReconnecting = !!sessionId;

        // Send resize message to backend
        const sendResize = () => {
            if (ws.readyState === WebSocket.OPEN) {
                const dimensions = fitAddon.proposeDimensions();
                if (dimensions) {
                    ws.send(JSON.stringify({
                        type: 'resize',
                        cols: dimensions.cols,
                        rows: dimensions.rows,
                    }));
                }
            }
        };

        ws.onopen = () => {
            setConnected(true);
            // Send initial size after connection
            setTimeout(() => {
                fitAddon.fit();
                sendResize();
                // Run initial command only for new sessions (not reconnects)
                if (!isReconnecting && initialCommand) {
                    setTimeout(() => {
                        ws.send(initialCommand + '\n');
                    }, 300);
                }
            }, 100);
        };

        ws.onmessage = (event) => {
            if (event.data instanceof ArrayBuffer) {
                const decoder = new TextDecoder();
                xterm.write(decoder.decode(event.data));
            } else {
                // Text message - could be a control message (JSON) or plain text
                const text = event.data as string;
                try {
                    const msg = JSON.parse(text);
                    if (msg.type === 'session_id' && msg.session_id) {
                        onSessionIdRef.current?.(msg.session_id);
                        return;
                    }
                } catch {
                    // Not JSON, write as terminal output
                }
                xterm.write(text);
            }
        };

        ws.onerror = () => {
            xterm.writeln('\r\n\x1b[31mConnection error\x1b[0m');
            setConnected(false);
        };

        ws.onclose = () => {
            xterm.writeln('\r\n\x1b[33mConnection closed\x1b[0m');
            setConnected(false);
        };

        // Handle user input
        xterm.onData((data) => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(data);
            }
        });

        // Handle resize
        const handleResize = () => {
            if (fitAddonRef.current) {
                fitAddonRef.current.fit();
                sendResize();
            }
        };
        window.addEventListener('resize', handleResize);

        // Focus terminal
        xterm.focus();

        return () => {
            window.removeEventListener('resize', handleResize);
            if (ws.readyState === WebSocket.OPEN) {
                ws.close();
            }
            wsRef.current = null;
            xterm.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isActive, theme, fontSize, fontFamily, cwd, name, initialCommand, sessionId]);

    // Fit terminal when container size changes
    useEffect(() => {
        if (isActive && fitAddonRef.current) {
            // Small delay to ensure container is rendered
            const timer = setTimeout(() => {
                fitAddonRef.current?.fit();
            }, 100);
            return () => clearTimeout(timer);
        }
    }, [isActive]);

    const sendKey = (key: string) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(key);
            xtermRef.current?.focus();
        }
    };

    const focus = () => {
        xtermRef.current?.focus();
    };

    return {
        terminalRef,
        connected,
        sendKey,
        focus,
    };
}
