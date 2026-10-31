import { useEffect, useState } from 'react';
import type { BrowserMode, BrowserTab, SessionInfo } from './browserWire';
import { presetLabel, windowTabs } from './browserView';
import { BrowserAddressEditor } from './BrowserAddressEditor';
import { useAddressVisualViewport } from './useAddressVisualViewport';
import { BrowserBookmarksSheetView } from './BrowserBookmarksSheetView';
import { BrowserCloseWindowDialog } from './BrowserCloseWindowDialog';
import { BrowserNewTabView } from './BrowserNewTabView';
import { BrowserTabList } from './BrowserTabList';
import { BrowserWindowsRail } from './BrowserWindowsRail';
import {
    nextRailOpen,
    paneTabsFor,
    paneTitleFor,
    railInitialOpen,
    railWide,
    readRailStored,
    storeRailOpen,
} from './railOpen';
import { closeTargetFor } from './windowClose';
import type { WindowRow } from './windowClose';
import type { BookmarkDocument, BookmarkItem, BookmarkSource } from './browserBookmarks';
export type SheetView = 'main' | 'url' | 'newtab' | 'bookmarks';
interface BrowserSheetProps {
    open: boolean;
    view: SheetView;
    sessions: SessionInfo[];
    current: string;
    tabs: BrowserTab[];
    activeTabId: number;
    busy: boolean;
    readOnly: boolean;
    /** Extension advertises window-close; older builds can only list windows. */
    windowCloseSupported: boolean;
    /** True while the shared help card is open, so the `?` can report it. */
    helpOpen: boolean;
    newWindowError: string | null;
    preset: string;
    /** The viewport the window achieved, when the daemon has reported it. */
    viewport: { width: number; height: number } | null;
    presets: { name: string; width: number; height: number }[];
    mode: BrowserMode;
    onClose: () => void;
    onView: (view: SheetView) => void;
    onNewWindow: () => Promise<boolean>;
    /** Resolves to the server's refusal, or null when the window closed. */
    onCloseWindow: (target: { window_id: number } | { session_id: string }) => Promise<string | null>;
    onTab: (tabId: number) => void;
    /** Activate a tab of any window: switches there first when needed. */
    onOpenTab: (sessionId: string, tabId: number) => void;
    onCloseTab: (tabId: number) => void;
    onCreateTab: (url: string) => Promise<boolean>;
    currentUrl?: string;
    bookmarked: boolean;
    bookmarks: BookmarkItem[];
    bookmarkDoc: BookmarkDocument | null;
    bookmarksLoading: boolean;
    bookmarksError: string | null;
    bookmarkSource: BookmarkSource;
    bookmarkProfile: string;
    localBookmarkCount: number | null;
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
}
/**
 * The one sheet: sessions, tabs, size, capture, URL.
 *
 * It deliberately stays open for Size and Capture — those actions change the
 * frame behind the sheet and a toast names the result, so closing would hide
 * the thing the person just asked for. Picking a tab or submitting a URL does
 * close it, because the page itself is the answer.
 */
export function BrowserSheet({
    open,
    view,
    sessions,
    current,
    tabs,
    activeTabId,
    busy,
    readOnly,
    windowCloseSupported,
    helpOpen,
    newWindowError,
    preset,
    viewport,
    presets,
    mode,
    onClose,
    onView,
    onNewWindow,
    onCloseWindow,
    onTab,
    onOpenTab,
    onCloseTab,
    onCreateTab,
    currentUrl,
    bookmarked,
    bookmarks,
    bookmarkDoc,
    bookmarksLoading,
    bookmarksError,
    bookmarkSource,
    bookmarkProfile,
    localBookmarkCount,
    onSaveBookmark,
    onRefreshBookmarks,
    onShowLocalBookmarks,
    onShowChromeBookmarks,
    onPreset,
    onMode,
    onReload,
    onNavigateTarget,
    onSelectAddressTab,
    onHelp,
}: BrowserSheetProps) {
    const sheetWrapRef = useAddressVisualViewport(open && view === 'url');
    const [reloading, setReloading] = useState(false);
    const [wasOpen, setWasOpen] = useState(open);
    const [closeSelection, setCloseSelection] = useState<{ scope: string; row: WindowRow; trigger: HTMLButtonElement } | null>(null);
    const [closing, setClosing] = useState(false);
    const [closeError, setCloseError] = useState('');
    const [newBusy, setNewBusy] = useState(false);
    // The window rail's open state is the person's stored intent, not sheet
    // state: it survives closes and reopens. First paint follows the stored
    // value, or the layout default (wide hosts start expanded).
    const [railOpen, setRailOpen] = useState(() => railInitialOpen(readRailStored(), railWide()));
    // Which window's tabs the right pane lists. Pure preview state: it follows
    // the connected window until the person previews another one.
    const [paneKey, setPaneKey] = useState(current);
    // Confirmation belongs to one open sheet, view and target window.
    // Scope mismatch resets it naturally without an effect that sets state on every poll.
    const scope = `${open ? 'open' : 'closed'}:${view}:${current}`;
    const confirming = open && closeSelection?.scope === scope;
    const visible = sessions.filter(
        (session) => session.attached || session.sessionless || session.id === current,
    );
    const currentSession = sessions.find((session) => session.id === current);
    const tabsLocked = Boolean(currentSession && (!currentSession.attached || currentSession.sessionless));
    const locked = readOnly || tabsLocked;
    const rows: WindowRow[] = visible.map((session) => ({
        key: session.id,
        sessionId: session.id,
        windowId: session.window_id,
        title: !session.attached && !session.sessionless ? 'Disconnected' : session.title?.trim() || session.id,
        attached: session.attached || Boolean(session.sessionless),
        tabCount: session.id === current ? tabs.length : windowTabs(session.tabs).length,
        current: session.id === current,
        sessionless: Boolean(session.sessionless),
    }));
    const liveCount = rows.filter((row) => row.attached).length;
    const currentRow =
        rows.find((row) => row.current) ??
        ({
            key: current || 'no-window',
            sessionId: current || undefined,
            title: 'No Chrome window',
            tabCount: tabs.length,
            current: true,
            attached: false,
            sessionless: tabsLocked,
        } satisfies WindowRow);
    const closeRow = confirming ? closeSelection.row : null;
    const closeTarget = closeRow ? closeTargetFor(closeRow) : null;

    // The right pane lists the previewed window's tabs; the left rail picks
    // which. Previewing never switches — tapping a tab there does.
    const paneRow = rows.find((row) => row.key === paneKey) ?? currentRow;
    const paneIsCurrent = paneRow.key === currentRow.key;
    const paneSession = sessions.find((session) => session.id === paneRow.sessionId);
    const paneTabs = paneTabsFor(paneSession, tabs, paneIsCurrent);
    const paneTitle = paneTitleFor(paneRow);

    // A closed sheet discards its view-specific state.
    if (wasOpen !== open) {
        setWasOpen(open);
        if (!open) { setCloseSelection(null); setPaneKey(current); }
    }

    // Switching windows anywhere (rail attach, a previewed tab, the parent)
    // drags the preview back to the window now being mirrored.
    if (paneKey !== current && !rows.some((row) => row.key === paneKey)) {
        setPaneKey(current);
    }
    useEffect(() => {
        setPaneKey(current);
    }, [current]);

    const toggleRail = () => {
        setCloseError('');
        setRailOpen((was) => {
            const next = nextRailOpen(was);
            storeRailOpen(next);
            return next;
        });
    };

    // The command is fire-and-forget; the brief spin is the only receipt the
    // sheet can give before the new frame arrives.
    useEffect(() => {
        if (!reloading) return;
        const timer = window.setTimeout(() => setReloading(false), 900);
        return () => window.clearTimeout(timer);
    }, [reloading]);

    const reload = () => {
        if (locked || reloading) return;
        setReloading(true);
        onReload();
    };

    const switchView = (next: SheetView) => onView(next);

    const openBookmark = async (item: BookmarkItem) => {
        if (locked || busy) return;
        if (await onCreateTab(item.url)) onClose();
    };

    const openLocalBookmarks = () => {
        onShowLocalBookmarks();
        switchView('bookmarks');
    };

    const achieved = viewport ?? presets.find((item) => item.name === preset) ?? null;
    const actualLabel = achieved ? `${Math.round(achieved.width)}×${Math.round(achieved.height)}` : '—';

    return (
        <div ref={sheetWrapRef} className={`bwp-sheetwrap${open ? ' is-open' : ''}`} aria-hidden={!open}>
            <div className="bwp-scrim" onClick={onClose} aria-hidden="true" />
            <section
                className="bwp-sheet"
                id="bwp-sheet"
                role="dialog"
                aria-modal="true"
                aria-label="Sessions and tabs"
                data-view={view}
            >
                <div className="bwp-sheetflow" inert={confirming ? true : undefined}>
                    <div className="bwp-grab" aria-hidden="true" />
                    {view === 'url' ? (
                    <BrowserAddressEditor
                        session={current} tabs={tabs} activeTabId={activeTabId} currentUrl={currentUrl} disabled={locked || busy}
                        bookmarkDoc={bookmarkDoc} bookmarksLoading={bookmarksLoading} bookmarksError={bookmarksError} bookmarkSource={bookmarkSource}
                        onShowLocal={onShowLocalBookmarks} onShowChrome={onShowChromeBookmarks} onBack={() => switchView('main')} onClose={onClose}
                        onSelectTab={onSelectAddressTab} onNavigate={onNavigateTarget}
                    />
                ) : view === 'newtab' ? (
                    <BrowserNewTabView
                        source={bookmarkSource}
                        localCount={localBookmarkCount}
                        doc={bookmarkDoc}
                        loading={bookmarksLoading}
                        error={bookmarksError}
                        disabled={locked || busy}
                        onBack={() => switchView('main')}
                        onClose={onClose}
                        onCreate={onCreateTab}
                        onOpenBookmark={openBookmark}
                        onOpenLocal={openLocalBookmarks}
                    />
                ) : view === 'bookmarks' ? (
                    <BrowserBookmarksSheetView
                        source={bookmarkSource}
                        profile={bookmarkProfile}
                        localCount={localBookmarkCount}
                        items={bookmarks}
                        doc={bookmarkDoc}
                        loading={bookmarksLoading}
                        error={bookmarksError}
                        currentUrl={currentUrl}
                        bookmarked={bookmarked}
                        disabled={locked || busy}
                        onBack={() => switchView('main')}
                        onClose={onClose}
                        onRefresh={onRefreshBookmarks}
                        onSave={onSaveBookmark}
                        onOpen={(item) => void openBookmark(item)}
                        onCreate={onCreateTab}
                        onShowLocal={onShowLocalBookmarks}
                        onShowChrome={onShowChromeBookmarks}
                    />
                ) : (
                    <>
                        <div className="bwp-shead">
                            <span className="bwp-kicker">Browser</span>
                            <span className="bwp-live">{liveCount} {liveCount === 1 ? 'window' : 'windows'}</span>
                            <div className="bwp-headacts">
                                <button
                                    type="button"
                                    className={`bwp-hbtn${reloading ? ' is-spin' : ''}`}
                                    onClick={reload}
                                    disabled={locked}
                                    aria-label="Reload the page"
                                    aria-busy={reloading}
                                    title="Reload the page (⌘R)"
                                >
                                    <span aria-hidden="true">↻</span>
                                </button>
                                <button
                                    type="button"
                                    className="bwp-hbtn"
                                    onClick={() => switchView('bookmarks')}
                                    aria-label="Bookmarks"
                                    title="Bookmarks"
                                >
                                    <span aria-hidden="true">☆</span>
                                </button>
                                <button
                                    type="button"
                                    className={`bwp-hbtn${helpOpen ? ' is-on' : ''}`}
                                    onClick={onHelp}
                                    aria-label="Help — gestures and keys"
                                    aria-expanded={helpOpen}
                                    aria-controls="bw-help"
                                    title="Help — gestures & keys"
                                >
                                    <span aria-hidden="true">?</span>
                                </button>
                                <button type="button" className="bwp-x" onClick={onClose} aria-label="Close">
                                    ×
                                </button>
                            </div>
                        </div>
                        <div className="bwp-duo">
                            <BrowserWindowsRail
                                rows={rows}
                                open={railOpen}
                                selectedKey={paneRow.key}
                                busy={busy}
                                readOnly={readOnly}
                                windowCloseSupported={windowCloseSupported}
                                newBusy={newBusy}
                                newWindowError={newWindowError}
                                onToggle={toggleRail}
                                onPreview={(row) => setPaneKey(row.key)}
                                onNew={() => {
                                    if (newBusy || busy) return;
                                    setNewBusy(true);
                                    void onNewWindow().then((ok) => {
                                        setNewBusy(false);
                                        if (ok) onClose();
                                    });
                                }}
                                onClose={(row, trigger) => {
                                    if (!closeTargetFor(row)) return;
                                    setCloseError('');
                                    setCloseSelection({ scope, row, trigger });
                                }}
                            />
                            <div className="bwp-panetabs">
                                <BrowserTabList
                                    visible={open}
                                    tabs={paneTabs}
                                    activeTabId={paneIsCurrent ? activeTabId : -1}
                                    busy={busy}
                                    readOnly={readOnly}
                                    tabsLocked={tabsLocked && paneIsCurrent}
                                    locked={locked}
                                    paneTitle={paneTitle}
                                    canClose={paneIsCurrent}
                                    canCreate={paneIsCurrent}
                                    onView={switchView}
                                    onTab={(tabId) => {
                                        if (paneIsCurrent) {
                                            onTab(tabId);
                                            return;
                                        }
                                        if (!paneRow.sessionId) return;
                                        // A previewed tab is the switch: go to that
                                        // window and activate the tab in one move.
                                        // Sessionless windows included: the direct
                                        // rail drives them by window id.
                                        onOpenTab(paneRow.sessionId, tabId);
                                    }}
                                    onCloseTab={onCloseTab}
                                />
                            </div>
                        </div>
                        <div className="bwp-foot">
                            <div className="bwp-row">
                                <span className="bwp-lbl">Size</span>
                                <div className="bwp-seg bwp-seg3" role="group" aria-label="Window size">
                                    {presets.map((item) => (
                                        <button
                                            key={item.name}
                                            type="button"
                                            className={item.name === preset ? 'is-on' : ''}
                                            aria-pressed={item.name === preset}
                                            onClick={() => onPreset(item.name)}
                                        >
                                            {presetLabel(item.name)}
                                        </button>
                                    ))}
                                </div>
                                <span className="bwp-actual" title="Viewport the Chrome window achieved">
                                    {actualLabel}
                                </span>
                            </div>
                            <div className="bwp-row">
                                <span className="bwp-lbl">Capture</span>
                                <div className="bwp-seg bwp-seg2" role="group" aria-label="Capture mode">
                                    <button
                                        type="button"
                                        className={mode === 'cast' ? 'is-on' : ''}
                                        aria-pressed={mode === 'cast'}
                                        onClick={() => onMode('cast')}
                                    >
                                        Cast
                                    </button>
                                    <button
                                        type="button"
                                        className={mode === 'shot' ? 'is-on' : ''}
                                        aria-pressed={mode === 'shot'}
                                        onClick={() => onMode('shot')}
                                    >
                                        Shot
                                    </button>
                                </div>
                                <span className="bwp-note">{mode === 'cast' ? 'auto · 6–12 fps' : 'auto · 1 fps'}</span>
                            </div>
                        </div>
                    </>
                )}
                </div>
                {closeRow && (
                    <BrowserCloseWindowDialog
                        title={closeRow.title}
                        sessionId={closeRow.sessionless ? undefined : closeRow.sessionId}
                        sessionless={closeRow.sessionless}
                        current={closeRow.current}
                        returnFocus={closeSelection?.trigger ?? null}
                        tabCount={closeRow.tabCount}
                        busy={closing}
                        error={closeError}
                        onCancel={() => {
                            if (closing) return;
                            setCloseSelection(null);
                            setCloseError('');
                        }}
                        onConfirm={() => {
                            if (closing || !closeTarget) return;
                            setClosing(true);
                            setCloseError('');
                            void onCloseWindow(closeTarget).then((failure) => {
                                setClosing(false);
                                if (failure) {
                                    setCloseError(failure);
                                    return;
                                }
                                setCloseSelection(null);
                                if (closeRow.current) { onClose(); }
                            });
                        }}
                    />
                )}
            </section>
        </div>
    );
}
