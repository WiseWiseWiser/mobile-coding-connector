import { MODE_LABEL } from './screenMode';
import type { ScreenMode } from './screenMode';

interface ModeChipProps {
    mode: ScreenMode;
    onToggle: () => void;
    className?: string;
}

/**
 * Shows which world the gestures land in and flips it on tap. REMOTE drives the
 * Mac; WEB keeps panning and zooming inside the browser. The label is the mode
 * you are in, not the one you would switch to — the state has to be readable at
 * a glance, and the two colours do the rest.
 */
export function ModeChip({ mode, onToggle, className }: ModeChipProps) {
    return (
        <button
            type="button"
            className={`screen-mode is-${mode}${className ? ` ${className}` : ''}`}
            onClick={onToggle}
            title={
                mode === 'remote'
                    ? 'Gestures drive the Mac. Tap for WEB mode to pan and zoom the picture.'
                    : 'Gestures pan and zoom the picture. Tap for REMOTE mode to drive the Mac.'
            }
        >
            {MODE_LABEL[mode]}
        </button>
    );
}
