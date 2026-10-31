import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { navigateTarget } from './browserApi';
import { useBrowserStream } from './useBrowserStream';
import { useBrowserSessions } from './useBrowserSessions';
import { useBrowserZoom } from './useBrowserZoom';
import { BrowserDesktopView } from './BrowserDesktopView';
import { useBrowserGestures } from './useBrowserGestures';
import { BrowserZoomer } from './BrowserZoomer';
import { BrowserPhoneView } from './BrowserPhoneView';
import { BrowserHelp } from './BrowserHelp';
import { BrowserTabCloseDialog } from './BrowserTabCloseDialog';
import { BrowserOffline } from './BrowserOffline';
import { useBrowserStartup } from './useBrowserStartup';
import { useBrowserTabClose } from './useBrowserTabClose';
import { BrowserOpenPalette } from './BrowserOpenPalette';
import { useBrowserBookmarks } from './useBrowserBookmarks';
import { useBrowserComposer } from './useBrowserComposer';
import type { SheetView } from './BrowserSheet';
import { useIsPhone } from './useIsPhone';
import { nextPhoneZoom, prevPhoneZoom } from './phoneZoom';
import { presetLabel, transformStyle, windowTabs } from './browserView';
import type { BrowserMode, Lens } from './browserWire';
import './browser.css';

const PRINTABLE = /^.$/u;
const TOAST_MS = 2800;
const PHONE_TOAST_MS = 2200;

/**
 * /browser — watch and drive a real Chrome window through browser-agent.
 *
 * Why this exists: a locked Mac cannot mirror its screen (synthetic HID and
 * screen capture are both denied), but Chrome still renders pages and the
 * browser-agent extension still drives them over CDP. That makes this page the
 * only way to finish a login, scan a QR code, or click through a flow while the
 * laptop is closed.
 *
 * Capture is on demand: the socket is the viewer, and the server stops taking
 * screenshots as soon as the last one leaves. Two lenses share the picture —
 * VIEW only pans and zooms, CTRL forwards touches to the page — so a glance at a
 * phone never presses anything.
 *
 * Under 900 px the page keeps the same picture but replaces the desktop cockpit
 * with two floating islands and one bottom sheet. The phone is the place this
 * page is actually used from, and a toolbar row's worth of controls on a 390 px
 * screen left no room for the page itself.
 */
export default function BrowserPage() {
    const navigate = useNavigate();
    const {
        status,
        connection,
        error,
        fallbackNote,
        dismissFallbackNote,
        clearError,
        attachStage,
        attachImage,
        attachCursor,
        setInput,
        send,
        navigate: go,
        refresh,
        transport,
        probe,
        pollInterval,
        switchTransport,
        retryWs,
        setPollInterval,
    } = useBrowserStream();
    const sessions = useBrowserSessions(!document.hidden);
    const startup = useBrowserStartup(sessions.refresh);
    const bookmarks = useBrowserBookmarks(true);
    const zoom = useBrowserZoom();
    const isPhone = useIsPhone();

    const [lens, setLens] = useState<Lens>('view');
    const [railOpen, setRailOpen] = useState(false);
    const [sheet, setSheet] = useState<'closed' | SheetView>('closed');
    const [helpOpen, setHelpOpen] = useState(false);
    const [dragging, setDragging] = useState(false);
    const [toast, setToast] = useState('');
    const [toastWarn, setToastWarn] = useState(false);
    const [keysVisible, setKeysVisible] = useState(() =>
        typeof window !== 'undefined' ? !!window.matchMedia?.('(pointer: coarse)').matches : false,
    );

    const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

    const showToast = (message: string, warn = false) => {
        if (toastTimer.current) clearTimeout(toastTimer.current);
        setToast(message);
        setToastWarn(warn);
        toastTimer.current = setTimeout(() => {
            setToast('');
            setToastWarn(false);
        }, TOAST_MS);
    };

    const { stageRefCb, imgRefCb } = useBrowserGestures({
        zoom,
        lens,
        send,
        attachStage,
        attachImage,
        setInput,
        setDragging,
    });

    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            // Help is a modal card: Escape dismisses only the card, and no key
            // may reach the page behind it.
            if (helpOpen) {
                if (event.key === 'Escape') setHelpOpen(false);
                return;
            }
            if (event.key === 'Escape') {
                setRailOpen(false);
                setSheet('closed');
                actions.closeComposer();
                return;
            }
            if (lens !== 'ctrl') return;
            const target = event.target as HTMLElement | null;
            if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return;
            if (event.metaKey || event.ctrlKey || event.altKey) return;
            if (event.key === 'Enter' || event.key === 'Tab' || event.key === 'Backspace' || event.key.startsWith('Arrow')) {
                const vk = event.key === 'Enter' ? 13 : event.key === 'Tab' ? 9 : event.key === 'Backspace' ? 8 : undefined;
                event.preventDefault();
                send({ type: 'key', key: event.key, code: event.code, vk });
                return;
            }
            if (PRINTABLE.test(event.key)) {
                event.preventDefault();
                send({ type: 'text', text: event.key });
            }
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [send, lens, helpOpen]);

    // A phone has no room for the rail: close it if the window shrinks.
    useEffect(() => {
        if (isPhone) setRailOpen(false);
        else setSheet('closed');
    }, [isPhone]);

    const sessionId = sessions.current || status?.session.id || '';
    const currentRow = sessions.sessions.find((row) => row.id === sessionId);
    const rowOffline = Boolean(currentRow && !currentRow.attached && !currentRow.sessionless);
    const sessionStatus = status?.session.id === sessionId ? status : null;
    const tabs = rowOffline ? [] : windowTabs(sessionStatus?.tabs ?? currentRow?.tabs);
    const activeTabId = rowOffline ? 0 : sessionStatus?.active_tab_id ?? currentRow?.active_tab_id ?? 0;
    const tabClose = useBrowserTabClose(sessions, (session, tabId) => {
        if (session === sessionId && tabId === activeTabId) zoom.reset();
        showToast('Tab closed.');
    }, (message) => showToast(message, true));
    const readOnly = Boolean(sessions.readOnly || status?.read_only);
    const sessionless = Boolean(currentRow?.sessionless);
    const currentUrl = rowOffline ? undefined : sessionStatus?.session.url ?? currentRow?.tabs?.find((tab) => tab.tab_id === activeTabId)?.url;
    const pageTitle = tabs.find((tab) => tab.tab_id === activeTabId)?.title ?? currentRow?.title;
    const mode: BrowserMode = status?.mode ?? 'cast';
    const connected = Boolean(status?.session.connected && status.session.window_id && status.session.id === sessionId && (!currentRow || currentRow.attached));
    const attaching = connection === 'open' && !connected;

    const actions = useBrowserComposer({
        sessions,
        bookmarks,
        sessionId,
        readOnly: readOnly || sessionless,
        currentUrl,
        pageTitle,
        notify: showToast,
        navigate: go,
        closeSheet: () => setSheet('closed'),
        resetZoom: zoom.reset,
        refresh,
    });

    // Zoom belongs to one page at a time. A tab switch, a session switch or a new
    // window all start fitted, so the picture never lands half off-screen.
    const lastTab = useRef(0);
    useEffect(() => {
        if (activeTabId && activeTabId !== lastTab.current) {
            lastTab.current = activeTabId;
            zoom.reset();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeTabId]);

    useEffect(() => {
        zoom.measure();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [status?.viewport?.width, status?.viewport?.height, sessionId]);

    const banner = toast || sessions.error || error;
    const phoneNotice = toast || fallbackNote || sessions.error || error;
    const phoneNoticeWarn = toast ? toastWarn : Boolean(fallbackNote || sessions.error || error);

    // Phone notices are ephemeral: a persistent strip would eat the picture, and
    // a stall is already obvious because the frame stops moving.
    useEffect(() => {
        if (!isPhone || !phoneNotice) return;
        const timer = window.setTimeout(() => {
            setToast('');
            setToastWarn(false);
            sessions.clearError();
            clearError();
            dismissFallbackNote();
        }, PHONE_TOAST_MS);
        return () => window.clearTimeout(timer);
        // The clearers are recreated every render; depending on them would restart
        // the timer on every status frame and the notice would never leave.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isPhone, phoneNotice]);

    const pickSession = async (id: string) => {
        setRailOpen(false);
        await sessions.switchTo(id);
        zoom.reset();
        refresh();
    };

    const openTab = async (session: string, tabId: number) => {
        setRailOpen(false);
        if (session !== sessionId) await sessions.switchTo(session);
        const result = await sessions.activate(session, tabId);
        if (result.error) showToast(result.error, true);
        else {
            zoom.reset();
            refresh();
        }
    };

    const closeTab = tabClose.close;

    const activateTab = async (tabId: number) => {
        if (tabId === activeTabId) return;
        const result = await sessions.activate(sessionId, tabId);
        if (result.error) showToast(result.error, true);
    };

    const newWindow = async () => {
        setRailOpen(false);
        const ok = await sessions.addSession();
        if (!ok) return false;
        zoom.reset();
        refresh();
        return true;
    };

    /** The sheet owns confirmation; this only reports the server's refusal. */
    const closeCurrentWindow = async (target: { window_id: number } | { session_id: string }) => {
        const result = await sessions.closeWindow(target);
        if (!result.ok) return result.error ?? 'cannot close that window';
        if (result.current && result.current !== sessionId) zoom.reset();
        refresh();
        showToast('Window closed.');
        return null;
    };

    const pickPreset = (preset: string) => {
        send({ type: 'preset', preset });
        const item = status?.presets?.find((entry) => entry.name === preset);
        showToast(`Requested ${presetLabel(preset)}${item ? ` · ${item.width}×${item.height}` : ''}.`);
    };

    const pickMode = (next: BrowserMode) => {
        send({ type: 'mode', mode: next });
        showToast(next === 'cast' ? 'Cast — live while a viewer is attached.' : 'Shot — 1 fps, plus a frame per tap.');
    };

    const stepZoom = (direction: 1 | -1) => {
        const target = direction > 0 ? nextPhoneZoom(zoom.view.zoom) : prevPhoneZoom(zoom.view.zoom);
        if (target !== null) zoom.zoomTo(target);
    };

    const fitZoom = () => {
        zoom.reset();
        showToast('Fit to screen.');
    };

    const reloadTab = () => {
        // A page reload is Cmd+R inside Chrome; `refresh` only asks the daemon for
        // another frame and would look identical on a page that is stuck.
        send({ type: 'key', key: 'r', code: 'KeyR', vk: 82, modifiers: 4 });
        showToast('Reloading the tab…');
    };

    const pickTab = (tabId: number) => {
        setSheet('closed');
        void activateTab(tabId);
    };

    const closeSheetTab = async (tabId: number) => {
        // Closing the last tab closes the window in Chrome; the rail fails over
        // to a free window on its own.
        await closeTab(sessionId, tabId);
    };

    const lensHint =
        lens === 'view'
            ? 'VIEW · drag to pan, double-tap or pinch to zoom'
            : mode === 'cast'
              ? 'CTRL · tap to click, drag to scroll, hold to drag'
              : 'CTRL · shot mode: about one frame per second';

    const stage = (
        <div className={`bw-stage${dragging ? ' is-dragging' : ''}`} ref={stageRefCb}>
            <img
                className="bw-frame"
                ref={imgRefCb}
                alt=""
                draggable={false}
                style={{ transform: transformStyle(zoom.transform) }}
                onLoad={zoom.measure}
            />
            <div className="bw-cursor" ref={attachCursor} />
            {dragging && <div className="bw-drag-tag">DRAG · lift to drop</div>}
            {!isPhone && <span className="bw-hint">{lensHint}</span>}
            {!isPhone && (
                <BrowserZoomer
                    canZoom={zoom.view.zoom < 6}
                    onZoomIn={() => zoom.step(1)}
                    onZoomOut={() => zoom.step(-1)}
                    onFit={zoom.reset}
                />
            )}
            {connection !== 'open' && (
                <div className="bw-overlay">
                    <span>{connection === 'connecting' ? 'Connecting…' : 'Reconnecting…'}</span>
                </div>
            )}
            {attaching && <BrowserOffline busy={startup.busy} stage={status?.startup} error={startup.error || (status?.startup === 'failed' ? status.error : '')} onStart={() => void startup.start()} />}
            {!attaching && connection === 'open' && status && !status.ok && (
                <div className="bw-overlay bw-overlay-error">
                    <span>{status.error || error || 'browser-agent is not available'}</span>
                </div>
            )}
        </div>
    );

    return (
        <div
            className={`bw-page lens-${lens}${railOpen ? ' is-rail' : ''}${isPhone ? ' bw-phone' : ''}${
                sheet !== 'closed' ? ' is-sheet' : ''
            }`}
        >
            {isPhone ? (
                <div className="bw-main">
                    <BrowserPhoneView
                        session={sessionId}
                        tabs={tabs}
                        activeTabId={activeTabId}
                        currentUrl={currentUrl}
                        connection={connection}
                        mode={mode}
                        lens={lens}
                        zoom={zoom.view.zoom}
                        keysVisible={keysVisible}
                        sheet={sheet}
                        sessions={sessions.sessions}
                        busy={sessions.busy}
                        readOnly={readOnly}
                        transport={transport}
                        probe={probe}
                        pollInterval={pollInterval}
                        onSwitchTransport={switchTransport}
                        onRetryWs={retryWs}
                        onPollInterval={setPollInterval}
                        windowCloseSupported={sessions.windowCloseSupported}
                        helpOpen={helpOpen}
                        newWindowError={sessions.newWindowError}
                        preset={status?.preset ?? 'mac'}
                        viewport={status?.viewport ?? null}
                        presets={status?.presets ?? []}
                        bookmarks={bookmarks.items}
                        bookmarkDoc={bookmarks.doc}
                        bookmarksLoading={bookmarks.loading}
                        bookmarksError={bookmarks.error}
                        bookmarkSource={bookmarks.source}
                        bookmarkProfile={bookmarks.profile}
                        localBookmarkCount={bookmarks.localCount}
                        bookmarked={actions.bookmarked}
                        notice={phoneNotice}
                        noticeWarn={phoneNoticeWarn}
                        onBack={() => navigate('/')}
                        onSheet={() => setSheet('main')}
                        onCloseSheet={() => setSheet('closed')}
                        onView={(view) => setSheet(view)}
                        onNewWindow={newWindow}
                        onCloseWindow={closeCurrentWindow}
                        onTab={pickTab}
                        onOpenTab={(id, tabId) => {
                            // A previewed tab is the switch: go to that window,
                            // activate the tab, and hand the page back.
                            setSheet('closed');
                            void openTab(id, tabId);
                        }}
                        onCloseTab={(tabId) => void closeSheetTab(tabId)}
                        onCreateTab={actions.createFromComposer}
                        onSaveBookmark={actions.saveCurrentPage}
                        onRefreshBookmarks={() => void bookmarks.refresh()}
                        onShowLocalBookmarks={bookmarks.showLocal}
                        onShowChromeBookmarks={bookmarks.showChrome}
                        onPreset={pickPreset}
                        onMode={pickMode}
                        onReload={reloadTab}
                        onNavigateTarget={async (session, tabId, url) => { try { await navigateTarget(session, tabId, url); zoom.reset(); return null; } catch (err) { return err instanceof Error ? err.message : String(err); } }}
                        onSelectAddressTab={async (session, tabId) => (await sessions.activate(session, tabId)).error ?? null}
                        onHelp={() => setHelpOpen(true)}
                        onLens={setLens}
                        onStep={stepZoom}
                        onFit={fitZoom}
                        onKeys={() => setKeysVisible((was) => !was)}
                        onText={(text) => send({ type: 'text', text })}
                        onKey={(key, code, vk) => send({ type: 'key', key, code, vk })}
                    >
                        {stage}
                    </BrowserPhoneView>
                </div>
            ) : (
                <BrowserDesktopView
                    status={status}
                    connection={connection}
                    session={sessionId}
                    tabs={tabs}
                    activeTabId={activeTabId}
                    readOnly={readOnly}
                    windowCloseSupported={sessions.windowCloseSupported}
                    sessionless={sessionless}
                    busy={sessions.busy}
                    loading={sessions.loading}
                    sessions={sessions.sessions}
                    daemon={sessions.daemon}
                    railOpen={railOpen}
                    lens={lens}
                    zoom={zoom.view.zoom}
                    keysVisible={keysVisible}
                    fallbackNote={fallbackNote}
                    banner={banner}
                    onHome={() => navigate('/')}
                    onRailToggle={() => setRailOpen((was) => !was)}
                    onRailClose={() => setRailOpen(false)}
                    onSelectSession={(id) => void pickSession(id)}
                    onOpenTab={(id, tabId) => void openTab(id, tabId)}
                    onCloseRailTab={(id, tabId) => void closeTab(id, tabId)}
                    onNewWindow={() => void newWindow()}
                    onCloseWindow={closeCurrentWindow}
                    onRefreshSessions={() => void sessions.refresh()}
                    onPreset={(preset) => send({ type: 'preset', preset })}
                    onMode={(next) => send({ type: 'mode', mode: next })}
                    onRefresh={refresh}
                    onUrl={actions.openAddress}
                    onActivateTab={(tabId) => void activateTab(tabId)}
                    onCloseTab={(id, tabId) => void closeTab(id, tabId)}
                    onNewTab={(draft) => actions.openComposer('newtab', draft)}
                    onBookmarks={() => actions.openComposer('bookmarks')}
                    bookmarked={actions.bookmarked}
                    onDismissNote={dismissFallbackNote}
                    onDismissBanner={() => {
                        setToast('');
                        sessions.clearError();
                        clearError();
                    }}
                    onKeysClose={() => setKeysVisible(false)}
                    onText={(text) => send({ type: 'text', text })}
                    onKey={(key, code, vk) => send({ type: 'key', key, code, vk })}
                    onLens={setLens}
                    onKeys={() => setKeysVisible((was) => !was)}
                    onHelp={() => setHelpOpen(true)}
                >
                    {stage}
                </BrowserDesktopView>
            )}

            {tabClose.pending && <BrowserTabCloseDialog pending={tabClose.pending} busy={tabClose.deciding} onKeep={() => void tabClose.choose(false)} onDiscard={() => void tabClose.choose(true)} onDismiss={tabClose.dismiss} />}
            {helpOpen && <BrowserHelp onClose={() => setHelpOpen(false)} />}
            {!isPhone && actions.composer && (
                <BrowserOpenPalette
                    key={actions.composer}
                    mode={actions.composer}
                    initialUrl={actions.seed}
                    currentUrl={currentUrl}
                    bookmarked={actions.bookmarked}
                    bookmarks={bookmarks.items}
                    bookmarkDoc={bookmarks.doc}
                    bookmarksLoading={bookmarks.loading}
                    bookmarksError={bookmarks.error}
                    bookmarkSource={bookmarks.source}
                    bookmarkProfile={bookmarks.profile}
                    localBookmarkCount={bookmarks.localCount}
                    disabled={!sessionId || readOnly || sessionless}
                    busy={sessions.busy}
                    onClose={actions.closeComposer}
                    onCreate={actions.createFromComposer}
                    onSaveCurrent={actions.saveCurrentPage}
                    onRefreshBookmarks={() => void bookmarks.refresh()}
                    onShowLocalBookmarks={bookmarks.showLocal}
                    onShowChromeBookmarks={bookmarks.showChrome}
                    onShowBookmarks={() => actions.openComposer('bookmarks')}
                />
            )}
        </div>
    );
}
