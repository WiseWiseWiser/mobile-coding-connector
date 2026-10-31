/**
 * The zoom detents the phone `−`/`+` buttons walk through.
 *
 * Pinch and double-tap stay continuous; the buttons snap so 100%, 150%, 200%
 * are always one press away instead of drifting by a multiplicative step.
 */
export const PHONE_ZOOMS = [1, 1.25, 1.5, 2, 2.5, 3, 4, 6] as const;

/** Tolerance for float zoom values that came from a pinch. */
const EPS = 0.001;

/** The next detent above `zoom`, or null when already at the top. */
export function nextPhoneZoom(zoom: number): number | null {
    return PHONE_ZOOMS.find((value) => value > zoom + EPS) ?? null;
}

/** The previous detent below `zoom`, or null when already fitted. */
export function prevPhoneZoom(zoom: number): number | null {
    for (let i = PHONE_ZOOMS.length - 1; i >= 0; i -= 1) {
        if (PHONE_ZOOMS[i] < zoom - EPS) return PHONE_ZOOMS[i];
    }
    return null;
}

/** Percent label shown on the `%` button. */
export function zoomPercent(zoom: number): string {
    return `${Math.round(zoom * 100)}%`;
}
