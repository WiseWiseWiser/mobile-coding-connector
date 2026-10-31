/**
 * Address-bar rules shared by the desktop omnibox, the phone sheet and the
 * new-tab composer.
 *
 * The client is deliberately not a URL normalizer: `localhost:3000` and a bare
 * search phrase are handed to the server exactly as typed, and the server keeps
 * the final say. What the client does do is refuse schemes the browser viewer
 * cannot mirror (`chrome://`, `file://`, `javascript:`, …) before a request is
 * sent, so the refusal can be shown next to the field instead of arriving as a
 * banner from the socket.
 */

export type AddressResult = { ok: true; target: string } | { ok: false; error: string };

const HOST_PORT = /^[a-z0-9.-]+:\d+(?:[/?#]|$)/i;
const EXPLICIT_SCHEME = /^([a-z][a-z0-9+.-]*):\/\//i;
const BARE_SCHEME = /^([a-z][a-z0-9+.-]*):/i;

/** Raw input → what the server should receive, or a sentence for the field. */
export function checkAddress(raw: string): AddressResult {
    const trimmed = raw.trim();
    if (!trimmed) return { ok: false, error: 'Enter an address first.' };
    if (/[\u0000-\u001f\u007f]/.test(trimmed)) {
        return { ok: false, error: 'That address contains control characters.' };
    }

    // `localhost:3000` looks like a scheme to a regex but is a host:port.
    const hostPort = HOST_PORT.test(trimmed);
    const explicit = hostPort ? null : EXPLICIT_SCHEME.exec(trimmed);
    const bare = hostPort || explicit ? null : BARE_SCHEME.exec(trimmed);
    const scheme = (explicit?.[1] ?? bare?.[1])?.toLowerCase();

    if (bare && (scheme === 'http' || scheme === 'https')) {
        return { ok: false, error: 'Use http:// or https:// before the address.' };
    }
    if (scheme && scheme !== 'http' && scheme !== 'https') {
        return {
            ok: false,
            error: `The viewer only opens http and https pages — ${scheme}:// is outside the mirror.`,
        };
    }

    if (explicit) {
        try {
            const parsed = new URL(trimmed);
            if (!parsed.hostname) return { ok: false, error: 'Enter a complete address, for example https://example.com.' };
        } catch {
            return { ok: false, error: 'That address is not valid.' };
        }
        return { ok: true, target: trimmed };
    }

    if (trimmed.startsWith('//')) return { ok: true, target: `https:${trimmed}` };
    return { ok: true, target: trimmed };
}

/** True when the omnibox should say Open rather than Search. */
export function looksLikeAddress(raw: string): boolean {
    const trimmed = raw.trim();
    if (!trimmed) return false;
    if (/^https?:\/\//i.test(trimmed)) return true;
    if (trimmed.startsWith('//')) return true;
    if (/^(localhost|127\.0\.0\.1)(:\d+)?(?:[/?#]|$)/i.test(trimmed)) return true;
    if (/\s/.test(trimmed)) return false;
    const head = trimmed.split('/')[0];
    return head.includes('.') || HOST_PORT.test(trimmed);
}

/** True for a page URL the mirror can show and a bookmark can store. */
export function isHttpURL(raw: string | null | undefined): boolean {
    if (!raw) return false;
    try {
        const parsed = new URL(raw);
        return (parsed.protocol === 'http:' || parsed.protocol === 'https:') && !!parsed.hostname;
    } catch {
        return false;
    }
}

/** `https://example.com/a` → `example.com/a`; falls back to the raw string. */
export function addressLabel(raw: string | null | undefined): string {
    if (!raw) return '';
    try {
        const parsed = new URL(raw);
        if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return raw;
        const path = parsed.pathname === '/' ? '' : parsed.pathname;
        return `${parsed.host}${path}${parsed.search}`;
    } catch {
        return raw.replace(/^https?:\/\//i, '').replace(/\/$/, '');
    }
}

/** Host for a toast, or the bare input while it is still being resolved. */
export function hostLabel(raw: string | null | undefined): string {
    if (!raw) return '';
    try {
        return new URL(raw).host || raw;
    } catch {
        return addressLabel(raw);
    }
}

/** A readable fallback title when saving the current page as a bookmark. */
export function bookmarkFallbackName(raw: string): string {
    try {
        const parsed = new URL(raw);
        const path = parsed.pathname === '/' ? '' : parsed.pathname;
        return `${parsed.host}${path}` || raw;
    } catch {
        return raw;
    }
}
