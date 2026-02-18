import { useState, useRef, useEffect, forwardRef, useImperativeHandle } from 'react';
import { useCurrent } from '../../../hooks/useCurrent';
import '@xterm/xterm/css/xterm.css';
import { PureTerminalView, type PureTerminalViewHandle } from '../../../components/pure-terminal/PureTerminalView';
import type { TerminalTheme } from '../../../hooks/usePureTerminal';
import { useTerminalTabs } from '../../../hooks/useTerminalTabs';
import { useV2Context } from '../../V2Context';
import './TerminalManager.css';

// Mobile-friendly dark theme
const v2Theme: TerminalTheme = {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#60a5fa',
    cursorAccent: '#0f172a',
    selectionBackground: '#334155',
    black: '#0f172a',
    red: '#ef4444',
    green: '#22c55e',
    yellow: '#eab308',
    blue: '#3b82f6',
    magenta: '#a855f7',
    cyan: '#06b6d4',
    white: '#f1f5f9',
    brightBlack: '#475569',
    brightRed: '#f87171',
    brightGreen: '#4ade80',
    brightYellow: '#facc15',
    brightBlue: '#60a5fa',
    brightMagenta: '#c084fc',
    brightCyan: '#22d3ee',
    brightWhite: '#ffffff',
};

// Re-export TerminalTab type for consumers
interface TerminalManagerProps {
    isVisible: boolean;
    /** Async function to load initial sessions. If provided, sessions will be loaded via this function. */
    loadSessions?: () => Promise<Array<{ id: string; name: string; cwd?: string }>>;
}


interface TerminalInstanceHandle {
    fit: () => void;
    focus: () => void;
}

// Ctrl mode state ref to share between component and callbacks
const useCtrlMode = () => {
    const ctrlModeRef = useRef(false);
    const [ctrlActive, setCtrlActive] = useState(false);
    
    const setCtrlMode = (active: boolean) => {
        ctrlModeRef.current = active;
        setCtrlActive(active);
    };
    
    return { ctrlModeRef, ctrlActive, setCtrlMode };
};

// Individual terminal instance component - uses PureTerminalView for the core terminal
const TerminalInstance = forwardRef<TerminalInstanceHandle, {
    id: string;
    isActive: boolean;
    cwd?: string;
    name?: string;
    initialCommand?: string;
    sessionId?: string;
    autoFocus?: boolean;
    onConnectionChange: (connected: boolean) => void;
    onReconnectRef: (reconnect: (() => void) | null) => void;
    onSessionId: (sessionId: string) => void;
    onCloseTab: () => void;
    onAutoFocusHandled: () => void;
}>(function TerminalInstance({
    id,
    isActive,
    cwd,
    name,
    initialCommand,
    sessionId,
    autoFocus,
    onConnectionChange,
    onReconnectRef,
    onSessionId,
    onCloseTab,
    onAutoFocusHandled,
}, ref) {
    const { ctrlModeRef, ctrlActive, setCtrlMode } = useCtrlMode();
    const autoFocusHandledRef = useRef(false);
    const shortcutsRef = useRef<HTMLDivElement>(null);
    const pureTerminalRef = useRef<PureTerminalViewHandle>(null);

    // iOS Safari keyboard detection using visualViewport API
    useEffect(() => {
        const visualViewport = (window as any).visualViewport;
        if (!visualViewport) {
            console.log('[iOS] No visualViewport API');
            return;
        }

        const handleResize = () => {
            const viewport = visualViewport;
            const windowHeight = window.innerHeight;
            const viewportHeight = viewport.height;
            const keyboardHeight = windowHeight - viewportHeight;
            
            console.log('[iOS] viewport resize:', { windowHeight, viewportHeight, keyboardHeight });
            
            // Update shortcuts position
            if (shortcutsRef.current) {
                if (keyboardHeight > 100) {
                    console.log('[iOS] Moving shortcuts up by:', keyboardHeight);
                    shortcutsRef.current.style.position = 'fixed';
                    shortcutsRef.current.style.bottom = `${keyboardHeight + (window.innerHeight > 700 ? 8 : 20)}px`;
                    shortcutsRef.current.style.left = '0';
                    shortcutsRef.current.style.right = '0';
                    shortcutsRef.current.style.zIndex = '9999';
                } else {
                    console.log('[iOS] Resetting shortcuts position');
                    shortcutsRef.current.style.position = '';
                    shortcutsRef.current.style.bottom = '';
                    shortcutsRef.current.style.left = '';
                    shortcutsRef.current.style.right = '';
                    shortcutsRef.current.style.zIndex = '';
                }
            }
            
            // Scroll terminal to keep cursor visible above keyboard
            if (keyboardHeight > 100) {
                const xtermViewport = document.querySelector('.terminal-instance.active .xterm-viewport') as HTMLElement;
                if (xtermViewport) {
                    console.log('[iOS] Scrolling terminal to bottom, scrollHeight:', xtermViewport.scrollHeight);
                    setTimeout(() => {
                        xtermViewport.scrollTop = xtermViewport.scrollHeight;
                        console.log('[iOS] After scroll, scrollTop:', xtermViewport.scrollTop);
                    }, 100);
                } else {
                    console.log('[iOS] No xtermViewport found');
                }
            }
        };

        visualViewport.addEventListener('resize', handleResize);
        // Also trigger once on mount
        setTimeout(handleResize, 500);
        return () => visualViewport.removeEventListener('resize', handleResize);
    }, []);

    // Handle connection changes
    const handleConnectionChange = (connected: boolean) => {
        onConnectionChange(connected);
        // Auto-focus when connection is established and autoFocus is requested
        if (connected && autoFocus && !autoFocusHandledRef.current) {
            autoFocusHandledRef.current = true;
            pureTerminalRef.current?.focus();
            onAutoFocusHandled();
        }
    };

    // Expose fit and focus methods to parent
    useImperativeHandle(ref, () => ({
        fit: () => setTimeout(() => pureTerminalRef.current?.fit(), 50),
        focus: () => pureTerminalRef.current?.focus(),
    }));

    // Report reconnect function to parent
    const onReconnectRefCb = useCurrent(onReconnectRef);
    if (isActive && pureTerminalRef.current) {
        onReconnectRefCb.current(() => pureTerminalRef.current?.reconnect());
    }

    const ctrlInputRef = useRef<HTMLInputElement | null>(null);
    const handleCtrl = () => {
        const next = !ctrlModeRef.current;
        setCtrlMode(next);
        // Focus the ctrl input field so user can type the next character
        if (next) {
            setTimeout(() => ctrlInputRef.current?.focus(), 0);
        }
    };
    const handleCtrlInput = (char: string) => {
        setCtrlMode(false);
        const charCode = char.toLowerCase().charCodeAt(0);
        if (charCode >= 97 && charCode <= 122) {
            // a-z → Ctrl+A (0x01) through Ctrl+Z (0x1A)
            pureTerminalRef.current?.sendKey(String.fromCharCode(charCode - 96));
        }
        pureTerminalRef.current?.focus();
    };
    const handleEsc = () => pureTerminalRef.current?.sendKey('\x1b');
    const handleCtrlC = () => pureTerminalRef.current?.sendKey('\x03');
    const handleCtrlA = () => pureTerminalRef.current?.sendKey('\x01');
    const handleCtrlR = () => pureTerminalRef.current?.sendKey('\x12');
    const handleCtrlL = () => pureTerminalRef.current?.sendKey('\x0c');
    const handleTab = () => pureTerminalRef.current?.sendKey('\t');
    const handleArrowUp = () => pureTerminalRef.current?.sendKey('\x1b[A');
    const handleArrowDown = () => pureTerminalRef.current?.sendKey('\x1b[B');
    const handleArrowLeft = () => pureTerminalRef.current?.sendKey('\x1b[D');
    const handleArrowRight = () => pureTerminalRef.current?.sendKey('\x1b[C');
    const handlePaste = async () => {
        try {
            const text = await navigator.clipboard.readText();
            if (text) pureTerminalRef.current?.sendKey(text);
        } catch (err) {
            console.error('Failed to paste from clipboard:', err);
        }
    };

    return (
        <div
            className={`terminal-instance ${isActive ? 'active' : ''}`}
            data-terminal-id={id}
        >
            <div className="terminal-instance-content">
                <PureTerminalView
                    ref={pureTerminalRef}
                    theme={v2Theme}
                    cwd={cwd}
                    name={name}
                    initialCommand={initialCommand}
                    sessionId={sessionId}
                    onSessionId={onSessionId}
                    onConnectionChange={handleConnectionChange}
                    onCloseRequest={onCloseTab}
                    autoFocus={autoFocus}
                />
                <div className="terminal-instance-shortcuts" ref={shortcutsRef}>
                    <button className="term-shortcut-btn" onClick={handleTab}>Tab</button>
                    <button className="term-shortcut-btn" onClick={handleArrowLeft}>←</button>
                    <button className="term-shortcut-btn" onClick={handleArrowRight}>→</button>
                    <button className="term-shortcut-btn" onClick={handleArrowUp}>↑</button>
                    <button className="term-shortcut-btn" onClick={handleArrowDown}>↓</button>
                    <button className={`term-shortcut-btn ${ctrlActive ? 'active' : ''}`} onClick={handleCtrl}>Ctrl</button>
                    {/* Hidden input to capture the next character when Ctrl mode is active */}
                    <input
                        ref={ctrlInputRef}
                        className="term-ctrl-hidden-input"
                        type="text"
                        maxLength={1}
                        autoCapitalize="none"
                        autoCorrect="off"
                        autoComplete="off"
                        onInput={(e) => {
                            const val = (e.target as HTMLInputElement).value;
                            if (val.length > 0) {
                                handleCtrlInput(val);
                                (e.target as HTMLInputElement).value = '';
                            }
                        }}
                        onBlur={() => {
                            setCtrlMode(false);
                        }}
                    />
                    <button className="term-shortcut-btn" onClick={handleEsc}>Esc</button>
                    <button className="term-shortcut-btn" onClick={handleCtrlC}>^C</button>
                    <button className="term-shortcut-btn" onClick={handleCtrlA}>^A</button>
                    <button className="term-shortcut-btn" onClick={handleCtrlR}>^R</button>
                    <button className="term-shortcut-btn" onClick={handleCtrlL}>^L</button>
                    <button className="term-shortcut-btn" onClick={handlePaste}>Paste</button>
                </div>
            </div>
        </div>
    );
});

export interface TerminalManagerHandle {
    /** Create a new terminal tab */
    createTab: (options?: { name?: string; cwd?: string; initialCommand?: string; autoFocus?: boolean }) => void;
    /** Open a new terminal tab (alias for createTab with specific params) */
    openTab: (name: string, cwd?: string, initialCommand?: string) => void;
    /** Refit the active terminal. Call this when the terminal container becomes visible. */
    fitActive: () => void;
}

export const TerminalManager = forwardRef<TerminalManagerHandle, TerminalManagerProps>(function TerminalManager({ isVisible, loadSessions }, ref) {
    const { currentProject } = useV2Context();
    const [zenMode, setZenMode] = useState(false);

    // Use the unified terminal tabs hook
    const {
        tabs,
        activeTabId,
        loading,
        error,
        setActiveTabId,
        createTab,
        closeTab,
        setTabSessionId,
        clearAutoFocus,
        ensureLoaded,
    } = useTerminalTabs({
        loadSessions,
        defaultCwd: currentProject?.dir,
    });

    // Trigger loading when component becomes visible
    useEffect(() => {
        if (isVisible) {
            ensureLoaded();
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isVisible]);

    const [activeConnected, setActiveConnected] = useState(false);
    const activeReconnectRef = useRef<(() => void) | null>(null);
    
    // Store refs to terminal instances for imperative fit calls
    const terminalRefsMap = useRef<Record<string, TerminalInstanceHandle | null>>({});

    // Ref for imperative handle
    const activeTabIdRef = useCurrent(activeTabId);

    useImperativeHandle(ref, () => ({
        createTab,
        openTab: (name: string, cwd?: string, initialCommand?: string) => {
            createTab({ name, cwd, initialCommand });
        },
        fitActive: () => {
            const activeId = activeTabIdRef.current;
            if (activeId) {
                terminalRefsMap.current[activeId]?.fit();
            }
        },
    }));

    const handleAddTab = () => {
        createTab();
    };

    // Show loading state while sessions are being fetched
    if (loading) {
        return (
            <div className="terminal-manager">
                <div className="terminal-manager-header">
                    <div className="terminal-tabs-container">
                        <div className="terminal-tab-item active">
                            <span className="terminal-tab-name">Loading...</span>
                        </div>
                    </div>
                    <div className="terminal-connection-status disconnected">
                        <span className="status-dot">○</span>
                        <span className="status-text">Loading</span>
                    </div>
                </div>
            </div>
        );
    }

    // Show empty state with "New Terminal" button when no tabs exist
    if (tabs.length === 0) {
        return (
            <div className="terminal-manager">
                <div className="terminal-manager-header">
                    <div className="terminal-tabs-container">
                        <button className="terminal-tab-add" onClick={handleAddTab} title="New Terminal">
                            + New Terminal
                        </button>
                    </div>
                    <div className="terminal-connection-status disconnected">
                        <span className="status-dot">○</span>
                        <span className="status-text">No terminals</span>
                    </div>
                </div>
                <div className="terminal-empty-state">
                    {error ? (
                        <p className="terminal-error">Error: {error}</p>
                    ) : (
                        <p>No terminal sessions</p>
                    )}
                    <button className="terminal-empty-state-btn" onClick={handleAddTab}>
                        Create New Terminal
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className={`terminal-manager ${zenMode ? 'zen-mode' : ''}`}>
            <div className="terminal-manager-header">
                <div className="terminal-tabs-container">
                    {tabs.map(tab => (
                        <div
                            key={tab.id}
                            className={`terminal-tab-item ${activeTabId === tab.id ? 'active' : ''}`}
                            onClick={() => {
                                setActiveTabId(tab.id);
                                // Refit terminal when switching tabs
                                terminalRefsMap.current[tab.id]?.fit();
                            }}
                        >
                            <span className="terminal-tab-name">{tab.name}</span>
                            {tabs.length > 1 && (
                                <button
                                    className="terminal-tab-close"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        closeTab(tab.id);
                                    }}
                                >
                                    ×
                                </button>
                            )}
                        </div>
                    ))}
                    <button className="terminal-tab-add" onClick={handleAddTab}>
                        +
                    </button>
                </div>
                {activeConnected ? (
                    <div className="terminal-connection-status connected">
                        <span className="status-dot">●</span>
                        <span className="status-text">Connected</span>
                    </div>
                ) : (
                    <button
                        className="terminal-connection-status disconnected clickable"
                        onClick={() => activeReconnectRef.current?.()}
                        title="Click to reconnect"
                    >
                        <span className="status-dot">○</span>
                        <span className="status-text">Reconnect</span>
                    </button>
                )}
                <button
                    className="terminal-zen-btn"
                    onClick={() => setZenMode(!zenMode)}
                    title={zenMode ? "Exit Zen Mode" : "Enter Zen Mode"}
                >
                    {zenMode ? 'Exit Zen' : 'Zen'}
                </button>
            </div>
            <div className="terminal-instances-container">
                {tabs.map(tab => (
                    <TerminalInstance
                        key={tab.id}
                        ref={(handle) => { terminalRefsMap.current[tab.id] = handle; }}
                        id={tab.id}
                        isActive={activeTabId === tab.id}
                        cwd={tab.cwd}
                        name={tab.name}
                        initialCommand={tab.initialCommand}
                        sessionId={tab.sessionId}
                        autoFocus={tab.autoFocus}
                        onConnectionChange={setActiveConnected}
                        onReconnectRef={(fn) => { if (activeTabId === tab.id) activeReconnectRef.current = fn; }}
                        onSessionId={(sid) => setTabSessionId(tab.id, sid)}
                        onCloseTab={() => closeTab(tab.id)}
                        onAutoFocusHandled={() => clearAutoFocus(tab.id)}
                    />
                ))}
            </div>
        </div>
    );
});
