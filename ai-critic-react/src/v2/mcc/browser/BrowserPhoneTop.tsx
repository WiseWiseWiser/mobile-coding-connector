import type { ReactNode } from 'react';
import type { BrowserMode } from './browserWire';
import { isHttpURL } from './browserAddress';

interface BrowserPhoneTopProps {
    session: string;
    sessionTitle?: string;
    sessionless: boolean;
    title: string;
    mode: BrowserMode;
    url?: string;
    readOnly: boolean;
    open: boolean;
    /** The poll badge rides the address banner: the transport is connection state. */
    badge?: ReactNode;
    onBack: () => void;
    onSheet: () => void;
    onAddress: () => void;
}

/** A single compact card: current tab, address, and an icon-only editor. */
export function BrowserPhoneTop({ session, sessionTitle, sessionless, title, mode, url, readOnly, open, badge, onBack, onSheet, onAddress }: BrowserPhoneTopProps) {
    const name = title?.trim() || sessionTitle?.trim() || session || 'New tab';
    const address = isHttpURL(url) ? url : '';
    const addressOff = sessionless || readOnly;
    return (
        <header className="bwp-top">
            <button type="button" className="bwp-icon bwp-back" onClick={onBack} aria-label="Back to Home" title="Back to Home">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M14.5 5 8 12l6.5 7" /></svg>
            </button>
            <button type="button" className="bwp-title" onClick={onSheet} aria-label={`Sessions and tabs: ${session}`} aria-expanded={open} aria-controls="bwp-sheet">
                <span className="bwp-sess"><i className={`bwp-dot is-${mode}`} aria-hidden="true" /><span className="bwp-sessname">{name}</span></span>
                <span className={`bwp-urltext${address ? '' : ' is-empty'}`} title={address || 'New tab'}>{address || 'New tab'}</span>
            </button>
            {badge}
            <button type="button" className="bwp-urlcap" onClick={onAddress} disabled={addressOff} title={addressOff ? 'Attach this window to open a page' : 'Edit address'} aria-label={addressOff ? 'Address unavailable until attached' : 'Edit address'}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M12 20h9" /><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L8 18l-4 1 1-4Z" /></svg>
            </button>
        </header>
    );
}
