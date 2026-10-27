import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useScreenSocket } from './useScreenSocket';
import { useHostKeyboard } from './useHostKeyboard';
import { MirrorView } from './MirrorView';
import { ControlBar } from './ControlBar';
import { KeyBar } from './KeyBar';
import { MAX_ZOOM, MIN_ZOOM } from './viewport';
import { ModeChip } from './ModeChip';
import { nextMode } from './screenMode';
import { canDriveHost } from './screenInput';
import type { ScreenMode } from './screenMode';
import { promptScreenPermission } from '../../../api/screen';
import type { ScreenStatus } from '../../../api/screen';
import './screen.css';

type FullscreenMode = 'native' | 'immersive' | null;

type FullscreenElement = HTMLDivElement & {
    webkitRequestFullscreen?: () => Promise<void> | void;
};

type FullscreenDocument = Document & {
    webkitFullscreenElement?: Element | null;
    webkitExitFullscreen?: () => Promise<void> | void;
};

function fullscreenElement(): Element | null {
    const doc = document as FullscreenDocument;
    return doc.fullscreenElement ?? doc.webkitFullscreenElement ?? null;
}

function isStandalone(): boolean {
    if (typeof window === 'undefined') return false;
    const installed = (navigator as Navigator & { standalone?: boolean }).standalone;
    return installed === true || !!window.matchMedia?.('(display-mode: standalone)').matches;
}

function PermissionBanner({ status }: { status: ScreenStatus | null }) {
    const [prompted, setPrompted] = useState(false);
    if (!status) return null;

    if (status.input.locked) {
        return (
            <div className="screen-banner screen-banner-warn">
                <span>
                    The host screen is locked, so remote input is paused. Control comes back
                    within a second of unlocking it.
                </span>
            </div>
        );
    }

    if (status.input.available) {
        return status.state === 'error' ? (
            <div className="screen-banner screen-banner-error">
                <span>Capture failed: {status.error}</span>
                {status.platform === 'darwin' && (
                    <button
                        className="screen-btn"
                        onClick={() => promptScreenPermission('screen').then(() => setPrompted(true)).catch(() => undefined)}
                    >
                        Open Screen Recording settings
                    </button>
                )}
            </div>
        ) : null;
    }

    return (
        <div className="screen-banner screen-banner-warn">
            <span>{status.input.reason || 'Input injection is unavailable, so this mirror is view-only.'}</span>
            {status.input.can_prompt && status.platform === 'darwin' && (
                <button
                    className="screen-btn"
                    onClick={() => promptScreenPermission('accessibility').then(() => setPrompted(true)).catch(() => undefined)}
                >
                    {prompted ? 'Opened — grant it, then reload' : 'Open Accessibility settings'}
                </button>
            )}
        </div>
    );
}

/**
 * /screen — a live, interactive mirror of the host desktop.
 *
 * Viewing needs nothing but the page: the stream starts as soon as the socket
 * connects. Typing and clicking require asking for control first.
 *
 * Two modes share the surface. REMOTE sends gestures to the Mac; WEB keeps them
 * in the browser so a corner of the picture can be zoomed into and read without
 * moving the host cursor or landing a click.
 */
export default function ScreenPage() {
    const navigate = useNavigate();
    const { screen, status, control, connection, error, inputError, view, attachCanvas, clearError } =
        useScreenSocket();

    const shellRef = useRef<HTMLDivElement>(null);
    const [busy, setBusy] = useState(false);
    const [dragging, setDragging] = useState(false);
    const [mode, setMode] = useState<ScreenMode>('web');
    const [fullscreen, setFullscreen] = useState<FullscreenMode>(null);
    const [fsHintDismissed, setFsHintDismissed] = useState(false);
    const [keysVisible, setKeysVisible] = useState(() => {
        const coarse = typeof window !== 'undefined' && window.matchMedia?.('(pointer: coarse)').matches;
        return !!coarse;
    });

    const immersive = fullscreen === 'immersive';
    useHostKeyboard(screen, control === 'active' && mode === 'remote');

    useEffect(() => {
        if (control === 'active') setBusy(false);
    }, [control]);

    // Taking control starts in REMOTE, letting go falls back to browsing.
    useEffect(() => {
        setMode(control === 'active' ? 'remote' : 'web');
    }, [control]);

    useEffect(() => {
        if (!error) return;
        const timer = window.setTimeout(() => clearError(), 6000);
        return () => window.clearTimeout(timer);
        // clearError is recreated per render; only the message should re-arm the timer.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error]);

    useEffect(() => {
        const release = () => screen.current?.releaseAll();
        window.addEventListener('beforeunload', release);
        return () => window.removeEventListener('beforeunload', release);
    }, [screen]);

    // Native fullscreen is only offered by some browsers, so the truth lives in
    // the event rather than in whatever we asked for.
    useEffect(() => {
        const doc = document as FullscreenDocument;
        const onChange = () => setFullscreen(fullscreenElement() ? 'native' : null);
        doc.addEventListener('fullscreenchange', onChange);
        doc.addEventListener('webkitfullscreenchange', onChange);
        return () => {
            doc.removeEventListener('fullscreenchange', onChange);
            doc.removeEventListener('webkitfullscreenchange', onChange);
        };
    }, []);

    // Escape leaves immersive mode, matching the native fullscreen behaviour.
    useEffect(() => {
        if (!immersive) return;
        const onKey = (event: KeyboardEvent) => {
            if (event.key === 'Escape') setFullscreen(null);
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [immersive]);

    const takeControl = () => {
        const remote = screen.current;
        if (!remote) return;
        setBusy(true);
        remote.requestControl();
        window.setTimeout(() => setBusy(false), 4000);
    };

    const releaseControl = () => {
        screen.current?.releaseControl();
    };

    const toggleFullscreen = () => {
        const shell = shellRef.current as FullscreenElement | null;
        const doc = document as FullscreenDocument;
        if (!shell) return;
        if (fullscreenElement()) {
            const exit = doc.exitFullscreen?.bind(doc) ?? doc.webkitExitFullscreen?.bind(doc);
            Promise.resolve(exit?.()).catch(() => setFullscreen(null));
            return;
        }
        const request = shell.requestFullscreen?.bind(shell) ?? shell.webkitRequestFullscreen?.bind(shell);
        if (request) {
            Promise.resolve(request()).catch(() => setFullscreen('immersive'));
            return;
        }
        // iOS Safari has no element fullscreen at all (only <video>). Hiding the
        // chrome is the closest thing, and it survives rotation.
        setFullscreen('immersive');
    };

    const starting = status?.state === 'starting';
    const stopped = !status || status.state === 'stopped';
    // Driving needs both the lock and a host that can take input; the composer
    // and the control button must agree on that, so both read this one flag.
    const canDrive = canDriveHost(status, control === 'active');
    const showHomeHint = immersive && !isStandalone() && !fsHintDismissed;
    const hint =
        control === 'locked'
            ? 'Someone else is driving. Take over to interact.'
            : mode === 'web'
              ? 'Browsing: drag to pan, pinch or double-tap to zoom. Take control to click or send text.'
              : 'Click the screen to take control.';

    return (
        <div className={`screen-page screen-page-${control}${immersive ? ' screen-page-immersive' : ''}`} ref={shellRef}>
            {!immersive && (
                <ControlBar
                    status={status}
                    control={control}
                    connection={connection}
                    onTake={takeControl}
                    onRelease={releaseControl}
                    onBack={() => navigate('/')}
                    onToggleFullscreen={toggleFullscreen}
                    keysVisible={keysVisible}
                    onToggleKeys={() => setKeysVisible((v) => !v)}
                    busy={busy}
                    mode={mode}
                    onToggleMode={() => setMode((m) => nextMode(m))}
                />
            )}
            {immersive && (
                <div className="screen-immersive-bar">
                    <button className="screen-icon-btn" onClick={() => navigate('/')} aria-label="back">
                        ‹
                    </button>
                    <ModeChip mode={mode} onToggle={() => setMode((m) => nextMode(m))} />
                    <span className="screen-immersive-title">
                        {control === 'active' ? 'driving the Mac' : 'view only'}
                    </span>
                    <button className="screen-icon-btn" onClick={() => setFullscreen(null)} aria-label="leave fullscreen">
                        ✕
                    </button>
                </div>
            )}
            <div className="screen-stage">
                <MirrorView
                    screen={screen}
                    attachCanvas={attachCanvas}
                    active={control === 'active'}
                    mode={mode}
                    onRequestControl={takeControl}
                    onDragChange={setDragging}
                />
                {control === 'active' && mode === 'remote' && (
                    <div
                        className={dragging ? 'screen-frame screen-frame-drag' : 'screen-frame'}
                        aria-hidden
                    />
                )}
                {dragging && <div className="screen-drag-badge">DRAG · lift to drop</div>}
                {starting && <div className="screen-overlay">starting capture…</div>}
                {status?.input.locked && (
                    <div className="screen-overlay screen-overlay-locked">
                        <span>
                            The Mac&rsquo;s screen is locked, so this image is the login window and input is
                            paused. Unlock the Mac to carry on.
                        </span>
                    </div>
                )}
                {stopped && connection === 'open' && <div className="screen-overlay">screen streaming is off</div>}
                {connection === 'closed' && <div className="screen-overlay">disconnected — retrying…</div>}
                <div className="screen-zoom">
                    <button
                        className="screen-zoom-btn"
                        onClick={() => screen.current?.zoomBy(-1)}
                        disabled={view.zoom <= MIN_ZOOM}
                        aria-label="zoom out"
                    >
                        −
                    </button>
                    <span className="screen-zoom-label">{view.zoom.toFixed(1)}×</span>
                    <button
                        className="screen-zoom-btn"
                        onClick={() => screen.current?.zoomBy(1)}
                        disabled={view.zoom >= MAX_ZOOM}
                        aria-label="zoom in"
                    >
                        ＋
                    </button>
                    <button
                        className="screen-zoom-btn"
                        onClick={() => screen.current?.fit()}
                        disabled={view.zoom <= MIN_ZOOM}
                        aria-label="fit to screen"
                    >
                        ⤢
                    </button>
                </div>
                {showHomeHint && (
                    <div className="screen-fs-hint">
                        <span>Add to Home Screen to hide the browser bars.</span>
                        <button className="screen-icon-btn" onClick={() => setFsHintDismissed(true)} aria-label="dismiss">
                            ✕
                        </button>
                    </div>
                )}
                {/* Anchored to the page these land on top of the on-screen key
                    bar on a phone, so both float over the picture instead. */}
                {!immersive && control !== 'active' && <div className="screen-hint">{hint}</div>}
                {error && (
                    <button className="screen-toast" onClick={clearError}>
                        {error}
                    </button>
                )}
            </div>
            {!immersive && <PermissionBanner status={status} />}
            <KeyBar
                screen={screen}
                platform={status?.platform ?? ''}
                visible={keysVisible}
                mode={mode}
                canDrive={canDrive}
                inputErrorSeq={inputError}
                onToggleMode={() => setMode((m) => nextMode(m))}
                onClose={() => setKeysVisible(false)}
            />
        </div>
    );
}
