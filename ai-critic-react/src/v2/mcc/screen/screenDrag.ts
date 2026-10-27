import type { ScreenMode } from './screenMode';

/** How long a finger must rest before it grabs the pointer. */
export const DRAG_HOLD_MS = 350;

/**
 * Whether a press may turn into a drag.
 *
 * Only one finger, only in REMOTE mode, only with control: in WEB mode the
 * picture pans locally instead, and without control there is no host cursor to
 * grab (the first tap asks for control instead).
 */
export function canStartDragHold(mode: ScreenMode, active: boolean, touches: number): boolean {
    return mode === 'remote' && active && touches === 1;
}

/** The two wire actions the latch needs; the caller owns the socket. */
export interface DragHost {
    /** Put a button down where the host cursor already hovers. */
    press(): void;
    /** Lift it wherever it ended up. */
    release(): void;
}

/**
 * Long-press to grab the host pointer, then drag until the finger lifts.
 *
 * A phone has no second button, so the drag has to be *latched*: resting a
 * finger on the picture presses the left button at the hovering cursor, and
 * every later move arrives as a dragged mouse event. The latch exists so the
 * lift can tell a drop from a tap, and so no path can leave the button pinned
 * on the host.
 */
export class DragLatch {
    private timer: ReturnType<typeof setTimeout> | null = null;
    private held = false;
    private readonly host: DragHost;
    /** Asked again when the timer fires: the finger may have wandered. */
    private readonly canGrab: () => boolean;
    private readonly onChange: (held: boolean) => void;
    private readonly holdMs: number;

    constructor(host: DragHost, canGrab: () => boolean, onChange: (held: boolean) => void, holdMs = DRAG_HOLD_MS) {
        this.host = host;
        this.canGrab = canGrab;
        this.onChange = onChange;
        this.holdMs = holdMs;
    }

    /** True while we own the button. */
    get active(): boolean {
        return this.held;
    }

    /** Start the hold timer for a finger that just landed. */
    arm(): void {
        this.cancel();
        if (this.held) return;
        this.timer = setTimeout(() => {
            this.timer = null;
            if (this.held || !this.canGrab()) return;
            this.host.press();
            this.held = true;
            this.onChange(true);
        }, this.holdMs);
    }

    /** Forget a pending grab that never happened. */
    cancel(): void {
        if (this.timer === null) return;
        clearTimeout(this.timer);
        this.timer = null;
    }

    /**
     * End the grab. `release` is false when the host already let go — losing the
     * lock makes the server drop all buttons, so there is nothing left to lift.
     */
    drop(release: boolean): void {
        this.cancel();
        if (!this.held) return;
        this.held = false;
        this.onChange(false);
        if (release) this.host.release();
    }
}
