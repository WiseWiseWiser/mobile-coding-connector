import { useRef, useEffect } from 'react';

/**
 * Hook that returns a ref that always contains the current value.
 * Useful for accessing the latest state in callbacks without adding
 * dependencies to useEffect or creating new callback references.
 */
export function useCurrent<T>(value: T): { current: T } {
    const ref = useRef(value);
    
    useEffect(() => {
        ref.current = value;
    }, [value]);
    
    return ref;
}
