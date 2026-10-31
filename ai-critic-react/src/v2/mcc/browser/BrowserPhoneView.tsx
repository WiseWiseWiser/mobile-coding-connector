import type { ReactNode } from 'react';
import { BrowserKeys } from './BrowserKeys';
import { BrowserPhoneBottom } from './BrowserPhoneBottom';
import { BrowserPhoneTop } from './BrowserPhoneTop';
import { BrowserProbeBanner, BrowserPollBadge } from './BrowserTransport';
import { BrowserSheet } from './BrowserSheet';
import type { SheetView } from './BrowserSheet';
import { tabLabel } from './browserView';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';
import type { PollInterval } from './browserPollStream';
import type { ProbeEvidence } from './browserStream';
import type {
    BrowserMode,
    BrowserTab,
    ConnectionState,
    Lens,
    SessionInfo,
} from './browserWire';
import type { StreamTransport } from './useBrowserStream';

export interface BrowserPhoneViewProps {
    session: string;
    tabs: BrowserTab[];
    activeTabId: number;
    currentUrl?: string;
    connection: ConnectionState;
    mode: BrowserMode;
    lens: Lens;
    zoom: number;
    keysVisible: boolean;
    sheet: 'closed' | SheetView;
    sessions: SessionInfo[];
    busy: boolean;
    readOnly: boolean;
    windowCloseSupported: boolean;
    helpOpen: boolean;
    newWindowError: string | null;
    preset: string;
    viewport: { width: number; height: number } | null;
    presets: { name: string; width: number; height: number }[];
    bookmarks: BookmarkItem[];
    bookmarkDoc: BookmarkDocument | null;
    bookmarksLoading: boolean;
    bookmarksError: string | null;
    bookmarkSource: BookmarkSource;
    bookmarkProfile: string;
    localBookmarkCount: number | null;
    bookmarked: boolean;
    notice: string | null;
    noticeWarn: boolean;
    /** Transport controls: the probe banner and the poll badge + menu. */
    transport: StreamTransport;
    probe: ProbeEvidence | null;
    pollInterval: number;
    onSwitchTransport: (next: StreamTransport) => void;
    onRetryWs: () => void;
    onPollInterval: (ms: PollInterval) => void;
    children: ReactNode;
    onBack: () => void;
    onSheet: () => void;
    onCloseSheet: () => void;
    onView: (view: SheetView) => void;
    onNewWindow: () => Promise<boolean>;
    onCloseWindow: (target: { window_id: number } | { session_id: string }) => Promise<string | null>;
    onTab: (tabId: number) => void;
    /** Activate a tab of any window: switches there first when needed. */
    onOpenTab: (sessionId: string, tabId: number) => void;
    onCloseTab: (tabId: number) => void;
    onCreateTab: (url: string) => Promise<boolean>;
    onSaveBookmark: () => Promise<boolean>;
    onRefreshBookmarks: () => void;
    onShowLocalBookmarks: () => void;
    onShowChromeBookmarks: () => void;
    onPreset: (preset: string) => void;
    onMode: (mode: BrowserMode) => void;
    onReload: () => void;
    onNavigateTarget: (session: string, tabId: number, url: string) => Promise<string | null>;
    onSelectAddressTab: (session: string, tabId: number) => Promise<string | null>;
    onHelp: () => void;
    onLens: (lens: Lens) => void;
    onStep: (direction: 1 | -1) => void;
    onFit: () => void;
    onKeys: () => void;
    onText: (text: string) => void;
    onKey: (key: string, code: string, vk?: number) => void;
}

/**
 * The phone shell: two islands, the picture, and one sheet. Everything here is
 * fixed-position chrome, so the same stage node can sit in the middle without
 * the desktop toolbar, rail or dock ever entering the tree on a phone.
 */
export function BrowserPhoneView(props: BrowserPhoneViewProps) {
    const activeTab = props.tabs.find((tab) => tab.tab_id === props.activeTabId);
    const title = activeTab
        ? tabLabel(activeTab)
        : props.connection === 'open'
          ? 'No tab'
          : 'Connecting…';
    const row = props.sessions.find((session) => session.id === props.session);

    return (
        <>
            <BrowserPhoneTop
                session={props.session}
                sessionTitle={row && !row.attached && !row.sessionless ? 'Disconnected' : row?.title}
                sessionless={Boolean(row?.sessionless)}
                title={title}
                mode={props.mode}
                url={props.currentUrl}
                readOnly={props.readOnly}
                open={props.sheet !== 'closed'}
                badge={
                    <BrowserPollBadge
                        transport={props.transport}
                        pollInterval={props.pollInterval}
                        onSwitchTransport={props.onSwitchTransport}
                        onPollInterval={props.onPollInterval}
                    />
                }
                onBack={props.onBack}
                onSheet={props.onSheet}
                onAddress={() => props.onView('url')}
            />

            <BrowserProbeBanner probe={props.probe} onSwitchTransport={props.onSwitchTransport} onRetryWs={props.onRetryWs} />

            {props.children}

            {props.keysVisible && props.lens === 'ctrl' && (
                <div className="bwp-tray">
                    <BrowserKeys
                        visible
                        onClose={props.onKeys}
                        onText={props.onText}
                        onKey={props.onKey}
                    />
                </div>
            )}

            <BrowserPhoneBottom
                lens={props.lens}
                zoom={props.zoom}
                keysVisible={props.keysVisible}
                onLens={props.onLens}
                onStep={props.onStep}
                onFit={props.onFit}
                onKeys={props.onKeys}
            />

            <BrowserSheet
                open={props.sheet !== 'closed'}
                view={props.sheet === 'closed' ? 'main' : props.sheet}
                sessions={props.sessions}
                current={props.session}
                tabs={props.tabs}
                activeTabId={props.activeTabId}
                busy={props.busy}
                readOnly={props.readOnly}
                windowCloseSupported={props.windowCloseSupported}
                helpOpen={props.helpOpen}
                newWindowError={props.newWindowError}
                preset={props.preset}
                viewport={props.viewport}
                presets={props.presets}
                mode={props.mode}
                currentUrl={props.currentUrl}
                bookmarked={props.bookmarked}
                bookmarks={props.bookmarks}
                bookmarkDoc={props.bookmarkDoc}
                bookmarksLoading={props.bookmarksLoading}
                bookmarksError={props.bookmarksError}
                bookmarkSource={props.bookmarkSource}
                bookmarkProfile={props.bookmarkProfile}
                localBookmarkCount={props.localBookmarkCount}
                onSaveBookmark={props.onSaveBookmark}
                onRefreshBookmarks={props.onRefreshBookmarks}
                onShowLocalBookmarks={props.onShowLocalBookmarks}
                onShowChromeBookmarks={props.onShowChromeBookmarks}
                onClose={props.onCloseSheet}
                onView={props.onView}
                onNewWindow={props.onNewWindow}
                onCloseWindow={props.onCloseWindow}
                onTab={props.onTab}
                onOpenTab={props.onOpenTab}
                onCloseTab={props.onCloseTab}
                onCreateTab={props.onCreateTab}
                onPreset={props.onPreset}
                onMode={props.onMode}
                onReload={props.onReload}
                onNavigateTarget={props.onNavigateTarget}
                onSelectAddressTab={props.onSelectAddressTab}
                onHelp={props.onHelp}
            />

            {props.notice && (
                <div className={`bwp-toast${props.noticeWarn ? ' warn' : ''}`} role="status" aria-live="polite">
                    {props.notice}
                </div>
            )}
        </>
    );
}
