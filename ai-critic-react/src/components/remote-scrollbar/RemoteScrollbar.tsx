import { useRef, useCallback, useState, useEffect, type CSSProperties } from 'react';
import './RemoteScrollbar.css';

export interface RemoteScrollbarProps {
    targetRef: React.RefObject<HTMLElement | null>;
    orientation?: 'horizontal' | 'vertical';
    thickness?: number;
    thumbColor?: string;
    trackColor?: string;
    thumbHoverColor?: string;
    className?: string;
    style?: CSSProperties;
    showTrack?: boolean;
    alwaysShow?: boolean;
}

export function RemoteScrollbar({
    targetRef,
    orientation = 'horizontal',
    thickness = 6,
    thumbColor = 'rgba(96, 165, 250, 0.8)',
    trackColor = 'rgba(30, 41, 59, 0.5)',
    thumbHoverColor = 'rgba(96, 165, 250, 1)',
    className = '',
    style = {},
    showTrack = true,
    alwaysShow = false,
}: RemoteScrollbarProps) {
    const [scrollPos, setScrollPos] = useState(0);
    const [maxScroll, setMaxScroll] = useState(0);
    const [thumbSize, setThumbSize] = useState(0);
    const [containerSize, setContainerSize] = useState(0);

    const isDragging = useRef(false);
    const touchStartPos = useRef(0);
    const scrollStartPos = useRef(0);

    const updateDimensions = useCallback(() => {
        const target = targetRef.current;
        if (!target) return;

        if (orientation === 'horizontal') {
            const scrollWidth = target.scrollWidth;
            const clientWidth = target.clientWidth;
            const max = scrollWidth - clientWidth;
            setMaxScroll(Math.max(0, max));
            setContainerSize(clientWidth);
            setThumbSize(clientWidth > 0 ? Math.max(20, (clientWidth / scrollWidth) * clientWidth) : 0);
            setScrollPos(target.scrollLeft);
        } else {
            const scrollHeight = target.scrollHeight;
            const clientHeight = target.clientHeight;
            const max = scrollHeight - clientHeight;
            setMaxScroll(Math.max(0, max));
            setContainerSize(clientHeight);
            setThumbSize(clientHeight > 0 ? Math.max(20, (clientHeight / scrollHeight) * clientHeight) : 0);
            setScrollPos(target.scrollTop);
        }
    }, [orientation, targetRef]);

    useEffect(() => {
        const target = targetRef.current;
        if (!target) return;

        updateDimensions();

        const resizeObserver = new ResizeObserver(() => {
            updateDimensions();
        });
        resizeObserver.observe(target);

        const handleScroll = () => {
            if (orientation === 'horizontal') {
                setScrollPos(target.scrollLeft);
            } else {
                setScrollPos(target.scrollTop);
            }
        };

        target.addEventListener('scroll', handleScroll, { passive: true });
        window.addEventListener('resize', updateDimensions);

        return () => {
            resizeObserver.disconnect();
            target.removeEventListener('scroll', handleScroll);
            window.removeEventListener('resize', updateDimensions);
        };
    }, [orientation, targetRef, updateDimensions]);

    const scrollTo = useCallback((newPos: number) => {
        const target = targetRef.current;
        if (!target) return;

        const clampedPos = Math.max(0, Math.min(maxScroll, newPos));
        if (orientation === 'horizontal') {
            target.scrollLeft = clampedPos;
        } else {
            target.scrollTop = clampedPos;
        }
        setScrollPos(clampedPos);
    }, [orientation, targetRef, maxScroll]);

    const handlePointerDown = useCallback((clientPos: number) => {
        isDragging.current = true;
        touchStartPos.current = clientPos;
        scrollStartPos.current = scrollPos;
        document.body.style.userSelect = 'none';
    }, [scrollPos]);

    const handlePointerMove = useCallback((clientPos: number) => {
        if (!isDragging.current) return;

        const target = targetRef.current;
        if (!target || maxScroll <= 0 || containerSize <= thumbSize) return;

        const delta = clientPos - touchStartPos.current;
        const ratio = maxScroll / (containerSize - thumbSize);
        const newPos = scrollStartPos.current + delta * ratio;
        scrollTo(newPos);
    }, [targetRef, maxScroll, containerSize, thumbSize, scrollTo]);

    const handlePointerUp = useCallback(() => {
        isDragging.current = false;
        document.body.style.userSelect = '';
    }, []);

    useEffect(() => {
        if (typeof window === 'undefined') return;

        const onMove = (e: MouseEvent | TouchEvent) => {
            if (isDragging.current) {
                const clientPos = 'touches' in e ? e.touches[0][orientation === 'horizontal' ? 'clientX' : 'clientY'] : e[orientation === 'horizontal' ? 'clientX' : 'clientY'];
                handlePointerMove(clientPos);
            }
        };

        const onUp = () => {
            if (isDragging.current) {
                handlePointerUp();
            }
        };

        window.addEventListener('mousemove', onMove, { passive: true });
        window.addEventListener('mouseup', onUp);
        window.addEventListener('touchmove', onMove, { passive: false });
        window.addEventListener('touchend', onUp);

        return () => {
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
            window.removeEventListener('touchmove', onMove);
            window.removeEventListener('touchend', onUp);
        };
    }, [orientation, handlePointerMove, handlePointerUp]);

    const handleTrackClick = useCallback((e: React.MouseEvent | React.TouchEvent) => {
        const target = targetRef.current;
        if (!target || maxScroll <= 0 || containerSize <= thumbSize) return;

        const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
        const clientPos = 'touches' in e ? e.touches[0][orientation === 'horizontal' ? 'clientX' : 'clientY'] : e[orientation === 'horizontal' ? 'clientX' : 'clientY'];
        const clickPos = orientation === 'horizontal' ? clientPos - rect.left : clientPos - rect.top;

        const currentThumbPos = maxScroll > 0 ? Math.max(0, Math.min(containerSize - thumbSize, scrollPos * (containerSize - thumbSize) / maxScroll)) : 0;
        if (clickPos >= currentThumbPos && clickPos <= currentThumbPos + thumbSize) return;

        let desiredThumbPos = clickPos - thumbSize / 2;
        desiredThumbPos = Math.max(0, Math.min(containerSize - thumbSize, desiredThumbPos));

        const newScrollPos = desiredThumbPos * maxScroll / (containerSize - thumbSize);
        scrollTo(newScrollPos);
    }, [orientation, targetRef, maxScroll, containerSize, thumbSize, scrollPos, scrollTo]);

    const isHorizontal = orientation === 'horizontal';
    const hasOverflow = maxScroll > 10;

    const trackStyle: CSSProperties = isHorizontal
        ? { width: '100%', height: thickness }
        : { width: thickness };

    const thumbPositionRaw = hasOverflow
        ? scrollPos * (containerSize - thumbSize) / maxScroll
        : 0;
    const thumbPosition = Math.max(0, Math.min(containerSize - thumbSize, thumbPositionRaw));

    const thumbStyle: CSSProperties = isHorizontal
        ? hasOverflow
            ? { width: thumbSize, transform: `translateX(${thumbPosition}px)`, backgroundColor: thumbColor }
            : { width: '100%', transform: `translateX(0px)`, backgroundColor: thumbColor }
        : hasOverflow
            ? { height: thumbSize, transform: `translateY(${thumbPosition}px)`, backgroundColor: thumbColor }
            : { height: '100%', transform: `translateY(0px)`, backgroundColor: thumbColor };

    if (!hasOverflow && !alwaysShow) {
        return null;
    }

    return (
        <div
            className={`remote-scrollbar remote-scrollbar--${orientation} ${className}`}
            style={{
                ...trackStyle,
                backgroundColor: showTrack ? trackColor : 'transparent',
                cursor: 'pointer',
                ...style,
            }}
            onMouseDown={(e) => {
                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                const clickPos = isHorizontal ? e.clientX - rect.left : e.clientY - rect.top;
                const thumbStart = thumbPosition;
                const thumbEnd = thumbPosition + thumbSize;
                if (clickPos >= thumbStart && clickPos <= thumbEnd) {
                    handlePointerDown(e[isHorizontal ? 'clientX' : 'clientY']);
                }
            }}
            onTouchStart={(e) => {
                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                const touch = e.touches[0];
                const clickPos = isHorizontal ? touch.clientX - rect.left : touch.clientY - rect.top;
                const thumbStart = thumbPosition;
                const thumbEnd = thumbPosition + thumbSize;
                if (clickPos >= thumbStart && clickPos <= thumbEnd) {
                    handlePointerDown(touch[isHorizontal ? 'clientX' : 'clientY']);
                }
            }}
            onClick={handleTrackClick}
        >
            {showTrack && (
                <div
                    className="remote-scrollbar__thumb"
                    style={thumbStyle}
                    onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = thumbHoverColor)}
                    onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = thumbColor)}
                />
            )}
        </div>
    );
}

export default RemoteScrollbar;
