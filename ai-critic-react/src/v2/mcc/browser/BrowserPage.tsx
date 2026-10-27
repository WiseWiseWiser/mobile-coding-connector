import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useBrowserStream } from './useBrowserStream';
import { useBrowserSessions } from './useBrowserSessions';
import { useBrowserZoom } from './useBrowserZoom';
import { BrowserGestures } from './browserGestures';
import type { GestureHost } from './browserGestures';
import { BrowserZoomPad } from './browserZoomPad';
import { BrowserToolbar } from './BrowserToolbar';
import { BrowserTabs } from './BrowserTabs';
import { BrowserRail } from './BrowserRail';
import { BrowserUrlBar } from './BrowserUrlBar';
import { BrowserKeys } from './BrowserKeys';
import { BrowserDock } from './BrowserDock';
import { BrowserZoomer } from './BrowserZoomer';
import { contentTabs, transformStyle } from './browserView';
import type { BrowserMode, Lens } from './browserWire';
import './browser.css';

const PRINTABLE = /^.$/u;
const TOAST_MS = 2800;

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
    } = useBrowserStream();
    const sessions = useBrowserSessions(!document.hidden);
    const zoom = useBrowserZoom();

    const [lens, setLens] = useState<Lens>('view');
    const [railOpen, setRailOpen] = useState(false);
    const [helpOpen, setHelpOpen] = useState(false);
    const [dragging, setDragging] = useState(false);
    const [toast, setToast] = useState('');
    const [keysVisible, setKeysVisible] = useState(() =>
        typeof window !== 'undefined' ? !!window.matchMedia?.('(pointer: coarse)').matches : false,
    );

    const stageEl = useRef<HTMLDivElement | null>(null);
    const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
    const sendRef = useRef(send);
    const lensRef = useRef<Lens>(lens);
    const zoomRef = useRef(zoom);
    sendRef.current = send;
    lensRef.current = lens;
    zoomRef.current = zoom;

    const showToast = (message: string) => {
        if (toastTimer.current) clearTimeout(toastTimer.current);
        setToast(message);
        toastTimer.current = setTimeout(() => setToast(''), TOAST_MS);
    };

    const [gestures] = useState(
        () =>
            new BrowserGestures({
                view: () => ({ geometry: zoomRef.current.geometry, view: zoomRef.current.view }),
                tap: (x, y) => sendRef.current({ type: 'tap', x, y, button: 1 }),
                down: (x, y) => sendRef.current({ type: 'down', x, y, button: 1 }),
                move: (x, y) => sendRef.current({ type: 'move', x, y, button: 1 }),
                up: (x, y) => sendRef.current({ type: 'up', x, y, button: 1 }),
                wheel: (x, y, dx, dy) =>
                    sendRef.current({ type: 'wheel', x, y, delta_x: dx, delta_y: dy }),
                onDragChange: setDragging,
            } satisfies GestureHost),
    );
    const gesturesRef = useRef<BrowserGestures | null>(null);
    gesturesRef.current = gestures;

    const [zoomPad] = useState(
        () =>
            new BrowserZoomPad({
                geometry: () => zoomRef.current.geometry,
                view: () => zoomRef.current.view,
                setView: (next) => zoomRef.current.setView(next),
                lens: () => lensRef.current,
                onMultiTouch: (active) => {
                    // A second finger means zoom, not a click: cancel whatever the
                    // first finger had started before the pad takes over.
                    if (active) {
                        gesturesRef.current?.standDown();
                        setDragging(false);
                    }
                },
                onDoubleTap: (anchor) => zoomRef.current.toggleAt(anchor),
            }),
    );

    // One stable ref callback: a new closure every render would detach and
    // reattach the gesture listeners on every frame.
    const [stageRefCb] = useState(() => (el: HTMLDivElement | null) => {
        stageEl.current = el;
        zoom.stageRef.current = el;
        attachStage(el);
    });
    const [imgRefCb] = useState(() => (el: HTMLImageElement | null) => {
        zoom.imgRef.current = el;
        attachImage(el);
    });

    useEffect(() => {
        const stage = stageEl.current;
        if (stage) zoomPad.attach(stage);
        setInput(lens === 'ctrl');
        return () => {
            zoomPad.detach();
            setInput(false);
        };
        // The stage element is stable for the life of the page.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [lens]);

    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setHelpOpen(false);
                setRailOpen(false);
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
    }, [send, lens]);

    const sessionId = sessions.current || status?.session.id || '';
    const currentRow = sessions.sessions.find((row) => row.id === sessionId);
    const tabs = contentTabs(status?.tabs ?? currentRow?.tabs);
    const activeTabId = status?.active_tab_id ?? currentRow?.active_tab_id ?? 0;
    const readOnly = Boolean(sessions.readOnly || status?.read_only);
    const mode: BrowserMode = status?.mode ?? 'cast';
    const connected = !!status?.session.connected;
    const attaching = connection === 'open' && !connected;

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
        if (result.error) showToast(result.error);
        else {
            zoom.reset();
            refresh();
        }
    };

    const closeTab = async (session: string, tabId: number) => {
        const wasActive = tabId === activeTabId;
        const result = await sessions.close(session, tabId);
        if (result.error) showToast(result.error);
        else if (wasActive) zoom.reset();
    };

    const createTab = async () => {
        const result = await sessions.create(sessionId);
        if (result.error) showToast(result.error);
        else {
            zoom.reset();
            refresh();
        }
    };

    const activateTab = async (tabId: number) => {
        if (tabId === activeTabId) return;
        const result = await sessions.activate(sessionId, tabId);
        if (result.error) showToast(result.error);
    };

    const newWindow = async () => {
        setRailOpen(false);
        await sessions.addSession();
        zoom.reset();
        refresh();
    };

    const banner = toast || sessions.error || error;
    const lensHint =
        lens === 'view'
            ? 'VIEW · drag to pan, double-tap or pinch to zoom'
            : mode === 'cast'
              ? 'CTRL · tap to click, drag to scroll, hold to drag'
              : 'CTRL · shot mode: about one frame per second';

    return (
        <div className={`bw-page lens-${lens}${railOpen ? ' is-rail' : ''}`}>
            <BrowserRail
                sessions={sessions.sessions}
                current={sessionId}
                daemon={sessions.daemon}
                loading={sessions.loading}
                busy={sessions.busy}
                open={railOpen}
                onSelect={pickSession}
                onOpenTab={openTab}
                onCloseTab={closeTab}
                onNew={newWindow}
                onRefresh={() => void sessions.refresh()}
            />
            {railOpen && <button type="button" className="bw-scrim" onClick={() => setRailOpen(false)} aria-label="Close sessions" />}

            <div className="bw-main">
                <BrowserToolbar
                    status={status}
                    connection={connection}
                    session={sessionId}
                    onBack={() => navigate('/')}
                    onRail={() => setRailOpen((was) => !was)}
                    onPreset={(preset) => send({ type: 'preset', preset })}
                    onMode={(next) => send({ type: 'mode', mode: next })}
                    onRefresh={refresh}
                />

                {readOnly && (
                    <div className="bw-ro-banner">
                        browser-agent {sessions.daemon?.version ?? '?'} cannot switch or close tabs — needs 1.0.20.
                        Watching and clicking still work.
                    </div>
                )}

                <BrowserTabs
                    tabs={tabs}
                    activeTabId={activeTabId}
                    session={sessionId}
                    busy={sessions.busy}
                    readOnly={readOnly}
                    onActivate={(tabId) => void activateTab(tabId)}
                    onClose={(tabId) => void closeTab(sessionId, tabId)}
                    onCreate={() => void createTab()}
                />

                <BrowserUrlBar url={status?.session.url} onNavigate={go} />

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
                    <span className="bw-hint">{lensHint}</span>
                    <BrowserZoomer
                        canZoom={zoom.view.zoom < 6}
                        onZoomIn={() => zoom.step(1)}
                        onZoomOut={() => zoom.step(-1)}
                        onFit={zoom.reset}
                    />
                    {connection !== 'open' && (
                        <div className="bw-overlay">
                            <span>{connection === 'connecting' ? 'Connecting…' : 'Reconnecting…'}</span>
                        </div>
                    )}
                    {attaching && (
                        <div className="bw-overlay">
                            <span>Attaching to Chrome…</span>
                            <small>
                                The browser-agent extension must be loaded in Chrome. The window opens on the Mac the
                                first time and takes a few seconds.
                            </small>
                        </div>
                    )}
                    {!attaching && connection === 'open' && status && !status.ok && (
                        <div className="bw-overlay bw-overlay-error">
                            <span>{status.error || error || 'browser-agent is not available'}</span>
                        </div>
                    )}
                </div>

                {fallbackNote && (
                    <button type="button" className="bw-note" onClick={dismissFallbackNote}>
                        {fallbackNote}
                    </button>
                )}
                {banner && (
                    <button
                        type="button"
                        className="bw-toast"
                        onClick={() => {
                            setToast('');
                            sessions.clearError();
                            clearError();
                        }}
                    >
                        {banner}
                    </button>
                )}

                <BrowserKeys
                    visible={keysVisible}
                    onClose={() => setKeysVisible(false)}
                    onText={(text) => send({ type: 'text', text })}
                    onKey={(key, code, vk) => send({ type: 'key', key, code, vk })}
                />
                <BrowserDock
                    lens={lens}
                    zoom={zoom.view.zoom}
                    readOnly={readOnly}
                    keysVisible={keysVisible}
                    onLens={setLens}
                    onKeys={() => setKeysVisible((was) => !was)}
                    onHelp={() => setHelpOpen(true)}
                />
            </div>

            {helpOpen && (
                <div className="bw-sheet" role="dialog" aria-label="Browser help" onClick={() => setHelpOpen(false)}>
                    <div className="bw-sheet-card" onClick={(event) => event.stopPropagation()}>
                        <h3>Two lenses, one window</h3>
                        <dl>
                            <dt>VIEW</dt>
                            <dd>Drag to pan, double-tap to zoom to 2.5×, pinch to zoom, nothing reaches the page.</dd>
                            <dt>CTRL</dt>
                            <dd>Tap to click, drag to scroll, hold 350 ms then drag, pinch to zoom, two fingers to pan.</dd>
                            <dt>Tabs</dt>
                            <dd>Pick a tab above, × closes it, + opens one. The last content tab cannot be closed.</dd>
                            <dt>Sessions</dt>
                            <dd>The rail lists every window the agent can drive; the dimmed ones reopen on click.</dd>
                            <dt>Keys</dt>
                            <dd>Hardware keys work in CTRL; ⌨ shows the on-screen keyboard for phones.</dd>
                        </dl>
                        <button type="button" className="bw-btn" onClick={() => setHelpOpen(false)}>
                            Close
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
