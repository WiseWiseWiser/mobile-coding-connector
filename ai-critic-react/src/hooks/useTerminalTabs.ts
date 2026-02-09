import { useState, useRef } from 'react';
import { deleteTerminalSession } from '../api/terminal';

export interface TerminalTab {
    id: string;
    name: string;
    cwd?: string;
    initialCommand?: string;
    /** Backend session ID, set after session is created or when restoring */
    sessionId?: string;
    /** Whether this tab should auto-focus when connected */
    autoFocus?: boolean;
}

interface UseTerminalTabsOptions {
    /** Async function to load initial sessions */
    loadSessions?: () => Promise<Array<{ id: string; name: string; cwd?: string }>>;
    /** Default working directory for new tabs */
    defaultCwd?: string;
}

interface UseTerminalTabsReturn {
    tabs: TerminalTab[];
    activeTabId: string | null;
    loading: boolean;
    error: string | null;
    setActiveTabId: (id: string) => void;
    createTab: (options?: { name?: string; cwd?: string; initialCommand?: string; autoFocus?: boolean }) => void;
    closeTab: (id: string) => void;
    setTabSessionId: (tabId: string, sessionId: string) => void;
    clearAutoFocus: (tabId: string) => void;
    /** Trigger loading sessions (idempotent — only loads once) */
    ensureLoaded: () => void;
}

/** Return the next unique "Terminal N" name that doesn't collide with existing names. */
function getNextTerminalName(existingNames: string[]): string {
    const nameSet = new Set(existingNames);
    let num = 1;
    while (nameSet.has(`Terminal ${num}`)) {
        num++;
    }
    return `Terminal ${num}`;
}

export function useTerminalTabs(options: UseTerminalTabsOptions = {}): UseTerminalTabsReturn {
    const { loadSessions, defaultCwd } = options;

    const [tabs, setTabs] = useState<TerminalTab[]>([]);
    const [activeTabId, setActiveTabId] = useState<string | null>(null);
    const [loading, setLoading] = useState(!!loadSessions);
    const [error, setError] = useState<string | null>(null);
    const loadedRef = useRef(false);

    const ensureLoaded = () => {
        if (loadedRef.current || !loadSessions) return;
        loadedRef.current = true;

        setLoading(true);
        loadSessions()
            .then(sessions => {
                if (sessions.length === 0) {
                    // Create a default terminal tab
                    const defaultTab: TerminalTab = {
                        id: `term-${Date.now()}`,
                        name: 'Terminal 1',
                        cwd: defaultCwd,
                    };
                    setTabs([defaultTab]);
                    setActiveTabId(defaultTab.id);
                } else {
                    const restored: TerminalTab[] = sessions.map(s => ({
                        id: `term-${s.id}`,
                        name: s.name,
                        cwd: s.cwd,
                        sessionId: s.id,
                    }));
                    setTabs(restored);
                    setActiveTabId(restored[0].id);
                }
                setLoading(false);
            })
            .catch(err => {
                setError(err?.message || 'Failed to load sessions');
                // Still create a default tab so the user isn't stuck
                const defaultTab: TerminalTab = {
                    id: `term-${Date.now()}`,
                    name: 'Terminal 1',
                    cwd: defaultCwd,
                };
                setTabs([defaultTab]);
                setActiveTabId(defaultTab.id);
                setLoading(false);
            });
    };

    // If no loadSessions, ensure we have at least one tab
    if (!loadSessions && !loadedRef.current) {
        loadedRef.current = true;
        if (tabs.length === 0) {
            const defaultTab: TerminalTab = {
                id: `term-${Date.now()}`,
                name: 'Terminal 1',
                cwd: defaultCwd,
            };
            setTabs([defaultTab]);
            setActiveTabId(defaultTab.id);
        }
    }

    const createTab = (opts?: { name?: string; cwd?: string; initialCommand?: string; autoFocus?: boolean }) => {
        const newId = `term-${Date.now()}`;
        const newName = opts?.name || getNextTerminalName(tabs.map(t => t.name));
        const newTab: TerminalTab = {
            id: newId,
            name: newName,
            cwd: opts?.cwd || defaultCwd,
            initialCommand: opts?.initialCommand,
            autoFocus: opts?.autoFocus,
        };
        setTabs(prev => [...prev, newTab]);
        setActiveTabId(newId);
    };

    const closeTab = (id: string) => {
        setTabs(prev => {
            const tab = prev.find(t => t.id === id);
            // Delete session on backend if it has one
            if (tab?.sessionId) {
                deleteTerminalSession(tab.sessionId).catch(() => { /* ignore */ });
            }

            const next = prev.filter(t => t.id !== id);
            if (next.length === 0) {
                // Create a new default tab when closing the last one
                const defaultTab: TerminalTab = {
                    id: `term-${Date.now()}`,
                    name: 'Terminal 1',
                    cwd: defaultCwd,
                };
                setActiveTabId(defaultTab.id);
                return [defaultTab];
            }

            // If we're closing the active tab, switch to an adjacent one
            if (activeTabId === id) {
                const closedIndex = prev.findIndex(t => t.id === id);
                const newActive = next[Math.min(closedIndex, next.length - 1)];
                setActiveTabId(newActive.id);
            }
            return next;
        });
    };

    const setTabSessionId = (tabId: string, sessionId: string) => {
        setTabs(prev => prev.map(t =>
            t.id === tabId ? { ...t, sessionId } : t
        ));
    };

    const clearAutoFocus = (tabId: string) => {
        setTabs(prev => prev.map(t =>
            t.id === tabId ? { ...t, autoFocus: false } : t
        ));
    };

    return {
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
    };
}
