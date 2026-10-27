import { useEffect } from 'react';
import type { RefObject } from 'react';
import type { RemoteScreen } from './RemoteScreen';

/** Keys the browser would otherwise consume before the page sees them. */
const BROWSER_KEYS = new Set(['Tab', 'F1', 'F3', 'F5', 'F6', 'F7', 'F11', 'F12']);

function isEditable(target: EventTarget | null): boolean {
    const el = target as HTMLElement | null;
    if (!el || !el.tagName) return false;
    return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT' || el.isContentEditable;
}

/**
 * Forwards physical key events to the host while this viewer owns control.
 * Printable characters travel as key codes so host modifiers/layouts apply;
 * anything the browser cannot describe as a code (IME, emoji) goes as text.
 */
export function useHostKeyboard(screen: RefObject<RemoteScreen | null>, active: boolean) {
    useEffect(() => {
        if (!active) return;

        const onKeyDown = (event: KeyboardEvent) => {
            if (isEditable(event.target)) return;
            const remote = screen.current;
            if (!remote) return;

            const combo = event.metaKey || event.ctrlKey || event.altKey;
            if (combo || BROWSER_KEYS.has(event.code)) {
                event.preventDefault();
            }
            if (event.isComposing) return;
            if (event.code && event.code !== 'Unidentified') {
                remote.key(event.code, true);
            } else if (event.key && event.key.length <= 8) {
                remote.text(event.key);
            }
        };

        const onKeyUp = (event: KeyboardEvent) => {
            if (isEditable(event.target)) return;
            const remote = screen.current;
            if (!remote) return;
            if (event.code && event.code !== 'Unidentified') {
                remote.key(event.code, false);
            }
        };

        // Losing focus with a modifier held is how you leave a stuck Cmd on the
        // host, so every exit path releases everything.
        const onBlur = () => screen.current?.releaseAll();
        const onVisibility = () => {
            if (document.visibilityState === 'hidden') screen.current?.releaseAll();
        };

        window.addEventListener('keydown', onKeyDown);
        window.addEventListener('keyup', onKeyUp);
        window.addEventListener('blur', onBlur);
        document.addEventListener('visibilitychange', onVisibility);
        const remote = screen.current;
        return () => {
            window.removeEventListener('keydown', onKeyDown);
            window.removeEventListener('keyup', onKeyUp);
            window.removeEventListener('blur', onBlur);
            document.removeEventListener('visibilitychange', onVisibility);
            remote?.releaseAll();
        };
    }, [screen, active]);
}
