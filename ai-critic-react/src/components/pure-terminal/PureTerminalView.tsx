import '@xterm/xterm/css/xterm.css';
import { useEffect, useRef, forwardRef, useImperativeHandle } from 'react';
import { usePureTerminal, type TerminalTheme } from '../../hooks/usePureTerminal';
import './PureTerminalView.css';

export interface PureTerminalViewProps {
    /** CSS class for the container */
    className?: string;
    /** Terminal theme - defaults to dark theme */
    theme?: TerminalTheme;
    /** Working directory for the terminal session */
    cwd?: string;
    /** Display name for the terminal session */
    name?: string;
    /** Command to run automatically after the terminal connects */
    initialCommand?: string;
    /** Existing session ID to reconnect to */
    sessionId?: string;
    /** Called when the backend assigns a session ID */
    onSessionId?: (sessionId: string) => void;
    /** Called when connection status changes */
    onConnectionChange?: (connected: boolean) => void;
    /** Called when user presses a key after connection closes */
    onCloseRequest?: () => void;
    /** Whether the terminal should auto-focus on mount */
    autoFocus?: boolean;
}

export interface PureTerminalViewHandle {
    /** Send a keystroke to the terminal */
    sendKey: (key: string) => void;
    /** Focus the terminal */
    focus: () => void;
    /** Reconnect the terminal */
    reconnect: () => void;
    /** Fit terminal to container */
    fit: () => void;
    /** Get connection status */
    connected: boolean;
}

/**
 * PureTerminalView - A reusable terminal component without tabs, quick input, or session management.
 * 
 * This is the core terminal view that can be used:
 * - Inside TerminalManager (with tabs and quick input)
 * - As a standalone terminal route
 * - In any other context where a terminal is needed
 * 
 * Features:
 * - xterm.js integration with WebSocket connection
 * - Mobile touch scroll support
 * - Auto-resize handling
 * - Connection status callbacks
 * - Imperative API via ref
 */
export const PureTerminalView = forwardRef<PureTerminalViewHandle, PureTerminalViewProps>(
    function PureTerminalView({
        className,
        theme,
        cwd,
        name,
        initialCommand,
        sessionId,
        onSessionId,
        onConnectionChange,
        onCloseRequest,
        autoFocus,
    }, ref) {
        const containerRef = useRef<HTMLDivElement>(null);
        const autoFocusHandledRef = useRef(false);

        const {
            terminalRef,
            connected,
            sendKey,
            focus,
            reconnect,
            fit,
        } = usePureTerminal({
            theme,
            cwd,
            name,
            initialCommand,
            sessionId,
            onSessionId,
            onConnectionChange,
            onCloseRequest,
        });

        // Handle auto-focus
        useEffect(() => {
            if (autoFocus && connected && !autoFocusHandledRef.current) {
                autoFocusHandledRef.current = true;
                focus();
            }
        }, [autoFocus, connected, focus]);

        // Expose imperative API
        useImperativeHandle(ref, () => ({
            sendKey,
            focus,
            reconnect,
            fit,
            connected,
        }), [sendKey, focus, reconnect, fit, connected]);

        return (
            <div 
                ref={containerRef}
                className={`pure-terminal-view ${className || ''}`}
            >
                <div className="pure-terminal-container" ref={terminalRef} />
            </div>
        );
    }
);

export default PureTerminalView;
