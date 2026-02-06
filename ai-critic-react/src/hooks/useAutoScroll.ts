import { useEffect, useRef } from 'react';

/**
 * Hook for smart auto-scrolling behavior:
 * - Automatically scrolls to the bottom when new content arrives (deps change)
 * - Yields scroll control to the user when they scroll up
 * - Resumes auto-scrolling when user scrolls back near the bottom
 *
 * @param deps - dependency array; auto-scroll triggers when these change
 * @param threshold - how close to the bottom (in px) the user must be to re-enable auto-scroll (default: 50)
 * @returns ref to attach to the scrollable container element
 */
export function useAutoScroll<T extends HTMLElement = HTMLDivElement>(
    deps: unknown[],
    threshold = 50
) {
    const containerRef = useRef<T | null>(null);
    const isAtBottomRef = useRef(true);

    // Track user scroll position
    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        const handleScroll = () => {
            const { scrollTop, scrollHeight, clientHeight } = container;
            isAtBottomRef.current = scrollHeight - scrollTop - clientHeight <= threshold;
        };

        container.addEventListener('scroll', handleScroll, { passive: true });
        return () => container.removeEventListener('scroll', handleScroll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [threshold]);

    // Auto-scroll when deps change, but only if user is at the bottom
    useEffect(() => {
        if (!isAtBottomRef.current) return;
        const container = containerRef.current;
        if (!container) return;

        // Use requestAnimationFrame to ensure DOM has updated
        requestAnimationFrame(() => {
            container.scrollTop = container.scrollHeight;
        });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);

    return containerRef;
}
