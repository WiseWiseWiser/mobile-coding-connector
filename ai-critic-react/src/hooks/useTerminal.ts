import { useEffect, useRef, useState } from 'react';
import type { MutableRefObject } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import type { TerminalTheme } from '../types/terminal';

export type { TerminalTheme };

export interface UseTerminalOptions {
    theme?: TerminalTheme;
    fontSize?: number;
    fontFamily?: string;
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
    } = options;

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

        // Connect to WebSocket terminal
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${protocol}//${window.location.host}/api/terminal`);
        ws.binaryType = 'arraybuffer';
        wsRef.current = ws;

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
            }, 100);
        };

        ws.onmessage = (event) => {
            if (event.data instanceof ArrayBuffer) {
                const decoder = new TextDecoder();
                xterm.write(decoder.decode(event.data));
            } else {
                xterm.write(event.data);
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
    }, [isActive, theme, fontSize, fontFamily]);

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
