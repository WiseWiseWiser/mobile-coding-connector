import type { BrowserTab } from './browserWire';

export interface BrowserTabsProps {
    tabs: BrowserTab[];
    activeTabId: number;
    session: string;
    busy: boolean;
    readOnly: boolean;
    onActivate: (tabId: number) => void;
    onClose: (tabId: number) => void;
    onCreate: () => void;
}

function label(tab: BrowserTab): string {
    const title = (tab.title ?? '').trim();
    if (title) return title;
    const host = (tab.url ?? '').replace(/^[a-z]+:\/\//i, '').split('/')[0];
    return host || 'New tab';
}

function initial(tab: BrowserTab): string {
    return (label(tab).slice(0, 1) || '·').toUpperCase();
}

/**
 * The tab strip for the session being streamed.
 *
 * Closing the last content tab is refused by the server. The session page is
 * filtered out before it gets here: it is plumbing, not a tab.
 */
export function BrowserTabs(props: BrowserTabsProps) {
    const disabled = props.busy || props.readOnly;
    return (
        <div className="bw-tabsrow">
            <div className="bw-tabs" role="tablist" aria-label={`Tabs in ${props.session}`}>
                {props.tabs.map((tab) => (
                    <div
                        key={tab.tab_id}
                        className={`bw-tab${tab.tab_id === props.activeTabId ? ' is-active' : ''}`}
                        data-tab={tab.tab_id}
                    >
                        <button
                            type="button"
                            role="tab"
                            aria-selected={tab.tab_id === props.activeTabId}
                            className="bw-tabopen"
                            onClick={() => props.onActivate(tab.tab_id)}
                            disabled={disabled}
                            title={label(tab)}
                        >
                            <span className="bw-fav">{initial(tab)}</span>
                            <span className="bw-tabname">{label(tab)}</span>
                        </button>
                        <button
                            type="button"
                            className="bw-tabx"
                            onClick={(event) => { event.currentTarget.focus({ preventScroll: true }); props.onClose(tab.tab_id); }}
                            disabled={disabled}
                            title="Close tab"
                            aria-label={`Close ${label(tab)}`}
                        >
                            ×
                        </button>
                    </div>
                ))}
            </div>
            <button
                type="button"
                className="bw-tabadd"
                onClick={props.onCreate}
                disabled={disabled}
                title="New tab"
                aria-label="New tab"
            >
                +
            </button>
        </div>
    );
}
