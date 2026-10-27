/**
 * Which world the fingers talk to.
 *
 *  - `remote` — every gesture is translated into host input (the default once
 *    you hold control);
 *  - `web`    — gestures pan and zoom the picture locally and nothing at all is
 *    sent to the Mac, so you can inspect a corner without clicking anything.
 */
export type ScreenMode = 'remote' | 'web';

export const MODE_LABEL: Record<ScreenMode, string> = {
    remote: 'REMOTE',
    web: 'WEB',
};

export function nextMode(mode: ScreenMode): ScreenMode {
    return mode === 'web' ? 'remote' : 'web';
}
