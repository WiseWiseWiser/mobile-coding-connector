import { useLayoutEffect, useRef } from 'react';

/** Keep the address sheet inside iOS Safari's visible area above the keyboard. */
export function useAddressVisualViewport(active: boolean) {
    const ref = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const element = ref.current;
        const viewport = window.visualViewport;
        if (!active || !element || !viewport) return;

        const update = () => {
            // VisualViewport is relative to the layout viewport; scroll matters
            // as well as resize when Safari pans to keep the focused input visible.
            element.style.top = `${viewport.offsetTop}px`;
            element.style.left = `${viewport.offsetLeft}px`;
            element.style.width = `${viewport.width}px`;
            element.style.height = `${viewport.height}px`;
            element.style.right = 'auto';
            element.style.bottom = 'auto';
        };
        update();
        viewport.addEventListener('resize', update);
        viewport.addEventListener('scroll', update);
        window.addEventListener('resize', update);
        return () => {
            viewport.removeEventListener('resize', update);
            viewport.removeEventListener('scroll', update);
            window.removeEventListener('resize', update);
            for (const property of ['top', 'left', 'width', 'height', 'right', 'bottom']) {
                element.style.removeProperty(property);
            }
        };
    }, [active]);
    return ref;
}
