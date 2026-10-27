import type { BrowserMode, BrowserStatus, ConnectionState } from './browserWire';

const PRESET_LABEL: Record<string, string> = {
    iphone: 'iPhone',
    ipad: 'iPad',
    mac: 'Mac',
};

const PRESET_TITLE: Record<string, string> = {
    iphone: 'Resize the Chrome window to an iPhone-sized viewport (390×844 requested).',
    ipad: 'Resize the Chrome window to an iPad-sized viewport (834×1194 requested).',
    mac: 'Restore the Chrome window to the size it had before any phone preset.',
};

interface BrowserToolbarProps {
    status: BrowserStatus | null;
    connection: ConnectionState;
    session: string;
    onBack: () => void;
    onRail: () => void;
    onPreset: (preset: string) => void;
    onMode: (mode: BrowserMode) => void;
    onRefresh: () => void;
}

function achievedLabel(status: BrowserStatus | null): string {
    if (!status) return '';
    const width = Math.round(status.viewport?.width ?? 0);
    const height = Math.round(status.viewport?.height ?? 0);
    if (!width || !height) return '';
    return `${width}×${height}`;
}

function clamped(status: BrowserStatus | null): boolean {
    if (!status) return false;
    const preset = status.presets?.find((item) => item.name === status.preset);
    if (!preset) return false;
    return (
        Math.abs((status.viewport?.width ?? preset.width) - preset.width) > 1 ||
        Math.abs((status.viewport?.height ?? preset.height) - preset.height) > 1
    );
}

/**
 * Connection state, the two capture modes and the window presets.
 *
 * The pill shows the viewport the window *achieved*, not the one that was asked
 * for: macOS clamps window width on small screens, and a lie here would make the
 * page look broken when it is the OS refusing.
 */
export function BrowserToolbar({
    status,
    connection,
    session,
    onBack,
    onRail,
    onPreset,
    onMode,
    onRefresh,
}: BrowserToolbarProps) {
    const live = connection === 'open' && !!status?.session.connected;
    const dotClass = connection !== 'open' ? 'is-warn' : live ? 'is-ok' : 'is-warn';
    const mode = status?.mode ?? 'cast';
    const viewport = achievedLabel(status);
    const presetTitle = status?.preset ? PRESET_TITLE[status.preset] : undefined;

    return (
        <div className="bw-toolbar">
            <div className="bw-toolbar-row">
                <button type="button" className="bw-icon" onClick={onBack} title="Back">
                    ‹
                </button>
                <span className={`bw-dot ${dotClass}`} />
                <span className="bw-title">BROWSER</span>
                {session && (
                    <span className="bw-sess" title={`Session ${session}`}>
                        {session}
                    </span>
                )}
                <span className={`bw-pill${clamped(status) ? ' is-clamped' : ''}`} title={presetTitle}>
                    {PRESET_LABEL[status?.preset ?? ''] ?? 'Mac'}
                    {viewport ? ` · ${viewport}` : ''}
                </span>
                <span className="bw-spacer" />
                <button type="button" className="bw-icon bw-railbtn" onClick={onRail} title="Sessions and tabs">
                    ☰
                </button>
                <button type="button" className="bw-icon" onClick={onRefresh} title="Request a fresh frame">
                    ↻
                </button>
            </div>
            <div className="bw-toolbar-row bw-chips">
                <button
                    type="button"
                    className={`bw-chip${mode === 'cast' ? ' is-on' : ''}`}
                    onClick={() => onMode('cast')}
                    title="Cast: poll frames as fast as the page changes. Best for watching."
                >
                    CAST
                </button>
                <button
                    type="button"
                    className={`bw-chip${mode === 'shot' ? ' is-on' : ''}`}
                    onClick={() => onMode('shot')}
                    title="Shot: one frame per second. Lighter on battery, slower to respond."
                >
                    SHOT
                </button>
                <span className="bw-sep" />
                {(status?.presets ?? []).map((preset) => (
                    <button
                        key={preset.name}
                        type="button"
                        className={`bw-chip${status?.preset === preset.name ? ' is-on' : ''}`}
                        onClick={() => onPreset(preset.name)}
                        title={`${PRESET_TITLE[preset.name] ?? ''} Requested ${preset.width}×${preset.height}.`}
                    >
                        {PRESET_LABEL[preset.name] ?? preset.name}
                    </button>
                ))}
            </div>
        </div>
    );
}
