import { useRevealCurrent } from './useRevealCurrent';
import type { BrowserTab } from './browserWire';
import type { SheetView } from './BrowserSheet';
import { tabLabel, tabMark } from './browserView';

export interface BrowserTabListProps {
    visible: boolean;
    tabs: BrowserTab[];
    activeTabId: number;
    busy: boolean;
    readOnly: boolean;
    tabsLocked: boolean;
    locked: boolean;
    /** Names the window whose tabs are listed (the two-pane header). */
    paneTitle?: string;
    /** False in a previewed window: closing someone else's tabs stays desktop-only. */
    canClose?: boolean;
    /** False in a previewed window: new tabs always land in the focused window. */
    canCreate?: boolean;
    onView: (view: SheetView) => void;
    onTab: (tabId: number) => void;
    onCloseTab: (tabId: number) => void;
}

/** The scrolling tab list. Only this list takes the sheet's leftover height. */
export function BrowserTabList(props: BrowserTabListProps) {
    const { tabs, activeTabId, busy, readOnly, tabsLocked, locked } = props;
    const canClose = props.canClose !== false;
    const canCreate = props.canCreate !== false;
    const listRef = useRevealCurrent<HTMLUListElement>(props.visible, '.bwp-trow');
    return (
        <>
            <div className="bwp-thead">
                <span className="bwp-kicker">Tabs</span>
                <span className="bwp-live">{tabs.length}</span>
                <span className="bwp-gap" />
                {props.paneTitle && <span className="bwp-panetitle">{props.paneTitle}</span>}
                <button
                    type="button"
                    className="bwp-mini"
                    onClick={() => props.onView('newtab')}
                    disabled={busy || readOnly || tabsLocked || !canCreate}
                    aria-label="New tab"
                    title={!canCreate ? 'New tabs open in the focused window' : tabsLocked ? 'Connect a Chrome window first' : 'New tab'}
                >
                    +
                </button>
            </div>
            <ul ref={listRef} className="bwp-tabs" aria-label={props.paneTitle ? `Tabs in ${props.paneTitle}` : 'Tabs in the current window'}>
                {tabs.length === 0 && <li className="bwp-empty">{tabsLocked ? 'No connected tabs.' : 'No tabs in this window yet.'}</li>}
                {tabs.map((tab) => {
                    const on = tab.tab_id === activeTabId;
                    return (
                        <li key={tab.tab_id} className={`bwp-trow${on ? ' is-on' : ''}`}>
                            <button
                                type="button"
                                className="bwp-topen"
                                aria-current={on ? 'true' : undefined}
                                onClick={() => props.onTab(tab.tab_id)}
                                disabled={busy || tabsLocked}
                            >
                                <span className="bwp-mono" aria-hidden="true">
                                    {tabMark(tab)}
                                </span>
                                <span className="bwp-tname">{tabLabel(tab)}</span>
                                {on && <span className="bwp-cur" aria-hidden="true" />}
                            </button>
                            {canClose && (
                                <button
                                    type="button"
                                    className="bwp-tclose"
                                    onClick={(event) => { event.currentTarget.focus({ preventScroll: true }); props.onCloseTab(tab.tab_id); }}
                                    disabled={busy || locked}
                                    aria-label={`Close ${tabLabel(tab)}`}
                                    title="Close tab"
                                >
                                    ×
                                </button>
                            )}
                        </li>
                    );
                })}
            </ul>
        </>
    );
}
