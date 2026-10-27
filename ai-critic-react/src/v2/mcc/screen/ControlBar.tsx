import type { ControlState, ConnectionState } from './RemoteScreen';
import { ModeChip } from './ModeChip';
import { controlPossible } from './screenInput';
import type { ScreenMode } from './screenMode';
import type { ScreenStatus } from '../../../api/screen';

interface ControlBarProps {
    status: ScreenStatus | null;
    control: ControlState;
    connection: ConnectionState;
    onTake: () => void;
    onRelease: () => void;
    onBack: () => void;
    onToggleFullscreen: () => void;
    keysVisible: boolean;
    onToggleKeys: () => void;
    busy: boolean;
    mode: ScreenMode;
    onToggleMode: () => void;
}

function controlLabel(control: ControlState): string {
    if (control === 'active') return 'ACTIVE — you are driving this desktop';
    if (control === 'locked') return 'LOCKED — someone else is driving';
    return 'VIEW ONLY';
}

/**
 * Status strip plus the control switch. The control state is deliberately loud:
 * a mirror that silently moves the host mouse is a trap.
 */
export function ControlBar({
    status, control, connection, onTake, onRelease, onBack,
    onToggleFullscreen, keysVisible, onToggleKeys, busy, mode, onToggleMode,
}: ControlBarProps) {
    const capture = status?.capture;
    const viewers = status?.viewers ?? 0;
    const locked = !!status?.input.locked;
    // Same predicate the composer uses, so the two never disagree about
    // whether this viewer can actually type into the host.
    const canControl = controlPossible(status);
    const holder = status?.control.holder;

    return (
        <header className={`screen-bar screen-bar-${control}`}>
            <button className="screen-icon-btn" onClick={onBack} title="Back to chat" aria-label="Back">
                ‹
            </button>
            <span className={`screen-dot screen-dot-${connection}`} aria-hidden />
            <div className="screen-bar-info">
                <strong>{controlLabel(control)}</strong>
                <span className="screen-bar-meta">
                    {connection !== 'open'
                        ? 'reconnecting…'
                        : capture && capture.width
                            ? `${capture.width}×${capture.height} · ${capture.fps} fps · ${viewers} viewer${viewers === 1 ? '' : 's'}`
                            : 'waiting for the first frame…'}
                    {holder && control !== 'active' ? ` · holder: ${holder}` : ''}
                </span>
            </div>
            <div className="screen-bar-actions">
                <ModeChip mode={mode} onToggle={onToggleMode} />
                {control === 'active' ? (
                    <button className="screen-btn screen-btn-release" onClick={onRelease}>
                        Release
                    </button>
                ) : (
                    <button
                        className="screen-btn screen-btn-primary"
                        onClick={onTake}
                        disabled={!canControl || busy || connection !== 'open'}
                        title={
                            locked
                                ? 'The host screen is locked'
                                : canControl
                                    ? 'Inject mouse and keyboard into the host'
                                    : 'Control is unavailable'
                        }
                    >
                        {control === 'locked' ? 'Take over' : 'Take control'}
                    </button>
                )}
                <button
                    className={`screen-icon-btn ${keysVisible ? 'is-on' : ''}`}
                    onClick={onToggleKeys}
                    title="On-screen keys"
                    aria-label="On-screen keys"
                >
                    ⌨
                </button>
                <button className="screen-icon-btn" onClick={onToggleFullscreen} title="Fullscreen" aria-label="Fullscreen">
                    ⛶
                </button>
            </div>
        </header>
    );
}
