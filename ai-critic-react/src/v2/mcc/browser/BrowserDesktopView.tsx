import type { ReactNode } from 'react';
import { BrowserDock } from './BrowserDock';
import { BrowserKeys } from './BrowserKeys';
import { BrowserRail } from './BrowserRail';
import { BrowserTabs } from './BrowserTabs';
import { BrowserToolbar } from './BrowserToolbar';
import { BrowserUrlBar } from './BrowserUrlBar';
import type {
    BrowserMode,
    BrowserStatus,
    BrowserTab,
    ConnectionState,
    DaemonStatus,
    Lens,
    SessionInfo,
} from './browserWire';

export interface BrowserDesktopViewProps {
    status: BrowserStatus | null;
    connection: ConnectionState;
    session: string;
    tabs: BrowserTab[];
    activeTabId: number;
    readOnly: boolean;
    windowCloseSupported: boolean;
    sessionless: boolean;
    busy: boolean;
    loading: boolean;
    sessions: SessionInfo[];
    daemon: DaemonStatus | null;
    railOpen: boolean;
    lens: Lens;
    zoom: number;
    keysVisible: boolean;
    fallbackNote: string | null;
    banner: string | null;
    children: ReactNode;
    onHome: () => void;
    onRailToggle: () => void;
    onRailClose: () => void;
    onSelectSession: (session: string) => void;
    onOpenTab: (session: string, tabId: number) => void;
    onCloseRailTab: (session: string, tabId: number) => void;
    onNewWindow: () => void;
    onCloseWindow: (target: { window_id: number } | { session_id: string }) => Promise<string | null>;
    onRefreshSessions: () => void;
    onPreset: (preset: string) => void;
    onMode: (mode: BrowserMode) => void;
    onRefresh: () => void;
    onUrl: (url: string) => void;
    onNewTab: (draft: string) => void;
    onBookmarks: () => void;
    bookmarked: boolean;
    onActivateTab: (tabId: number) => void;
    onCloseTab: (session: string, tabId: number) => void;
    onDismissNote: () => void;
    onDismissBanner: () => void;
    onKeysClose: () => void;
    onText: (text: string) => void;
    onKey: (key: string, code: string, vk?: number) => void;
    onLens: (lens: Lens) => void;
    onKeys: () => void;
    onHelp: () => void;
}

/** The desktop cockpit: rail, toolbar, tabs, URL bar, keys and dock. */
export function BrowserDesktopView(props: BrowserDesktopViewProps) {
    return (
        <>
            <BrowserRail
                sessions={props.sessions}
                current={props.session}
                daemon={props.daemon}
                loading={props.loading}
                busy={props.busy}
                readOnly={props.readOnly}
                windowCloseSupported={props.windowCloseSupported}
                open={props.railOpen}
                onSelect={props.onSelectSession}
                onOpenTab={props.onOpenTab}
                onCloseTab={props.onCloseRailTab}
                onNew={props.onNewWindow}
                onCloseWindow={props.onCloseWindow}
                onRefresh={props.onRefreshSessions}
            />
            {props.railOpen && (
                <button
                    type="button"
                    className="bw-scrim"
                    onClick={props.onRailClose}
                    aria-label="Close sessions"
                />
            )}

            <div className="bw-main">
                <BrowserToolbar
                    status={props.status}
                    connection={props.connection}
                    session={props.session}
                    onBack={props.onHome}
                    onRail={props.onRailToggle}
                    onPreset={props.onPreset}
                    onMode={props.onMode}
                    onRefresh={props.onRefresh}
                />

                {props.readOnly && !props.sessionless && (
                    <div className="bw-ro-banner">
                        browser-agent {props.daemon?.version ?? '?'} cannot switch or close tabs — needs 1.0.20.
                        Watching and clicking still work.
                    </div>
                )}

                <BrowserTabs
                    tabs={props.tabs}
                    activeTabId={props.activeTabId}
                    session={props.session}
                    busy={props.busy}
                    readOnly={props.readOnly || props.sessionless}
                    onActivate={props.onActivateTab}
                    onClose={(tabId) => props.onCloseTab(props.session, tabId)}
                    onCreate={() => props.onNewTab('')}
                />

                <BrowserUrlBar
                    url={props.status?.session.url}
                    disabled={!props.session || props.readOnly || props.sessionless}
                    bookmarked={props.bookmarked}
                    onNavigate={props.onUrl}
                    onNew={props.onNewTab}
                    onBookmarks={props.onBookmarks}
                />
                {props.children}

                {props.fallbackNote && (
                    <button type="button" className="bw-note" onClick={props.onDismissNote}>
                        {props.fallbackNote}
                    </button>
                )}
                {props.banner && (
                    <button type="button" className="bw-toast" onClick={props.onDismissBanner}>
                        {props.banner}
                    </button>
                )}

                <BrowserKeys
                    visible={props.keysVisible}
                    onClose={props.onKeysClose}
                    onText={props.onText}
                    onKey={props.onKey}
                />
                <BrowserDock
                    lens={props.lens}
                    zoom={props.zoom}
                    readOnly={props.readOnly}
                    keysVisible={props.keysVisible}
                    onLens={props.onLens}
                    onKeys={props.onKeys}
                    onHelp={props.onHelp}
                />
            </div>
        </>
    );
}
