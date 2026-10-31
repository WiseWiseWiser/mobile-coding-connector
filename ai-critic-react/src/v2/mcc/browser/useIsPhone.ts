import { useEffect, useState } from 'react';

/**
 * True when the phone chrome should replace the desktop rail.
 *
 * The breakpoint has to match the CSS exactly: the phone layout is not a
 * narrowed desktop, it is a different set of controls, so React and the
 * stylesheet must agree on where the swap happens.
 */
const PHONE_QUERY = '(max-width: 899px)';

export function useIsPhone(): boolean {
    const [phone, setPhone] = useState(() => window.matchMedia(PHONE_QUERY).matches);

    useEffect(() => {
        const query = window.matchMedia(PHONE_QUERY);
        const sync = () => setPhone(query.matches);
        sync();
        query.addEventListener('change', sync);
        return () => query.removeEventListener('change', sync);
    }, []);

    return phone;
}
