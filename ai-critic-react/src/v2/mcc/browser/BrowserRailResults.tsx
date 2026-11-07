import { matchCount, searchTokens } from './tabSearch';
import type { SearchGroup } from './tabSearch';
import { Marked } from './Marked';

export interface BrowserRailResultsProps {
    query: string;
    groups: SearchGroup[];
    /** Windows searched, so an empty answer can say how wide it looked. */
    windowCount: number;
    /** Jump to a tab, switching windows when the hit belongs to another one. */
    onPick: (group: SearchGroup, tabId: number) => void;
    disabled?: boolean;
}

function initialOf(title: string, url: string): string {
    const name = title.trim();
    if (name) return name.slice(0, 1).toUpperCase();
    const host = url.replace(/^[a-z]+:\/\//i, '').split('/')[0];
    return (host.slice(0, 1) || '·').toUpperCase();
}

/**
 * The rail's search answer: the same grouped-by-window list the phone draws,
 * with the window rows and tab rows the rail already uses. It replaces the tree
 * inside the body, so the field above it never moves.
 */
export function BrowserRailResults(props: BrowserRailResultsProps) {
    const tokens = searchTokens(props.query);
    const count = matchCount(props.groups);
    // Enter opens the first hit, so the mockup marks it as the row that will go.
    const lead = props.groups[0]?.hits[0];
    return (
        <>
            <p className="bw-rmeta">
                {count} {count === 1 ? 'tab' : 'tabs'} in {props.groups.length}{' '}
                {props.groups.length === 1 ? 'window' : 'windows'}
            </p>
            {count === 0 && (
                <div className="bw-rempty">
                    <span className="bw-remptyq">“{props.query.trim()}”</span>
                    <span>
                        No tab title or URL matches in {props.windowCount}{' '}
                        {props.windowCount === 1 ? 'window' : 'windows'}.
                    </span>
                </div>
            )}
            {props.groups.map((group) => (
                <section className="bw-rgroupwrap" key={group.key}>
                    <div className="bw-rgroup">
                        <span className="bw-rgtitle">{group.title}</span>
                        <span className="bw-rgmeta">{group.hits.length}</span>
                    </div>
                    {group.hits.map((hit) => (
                        <button
                            key={hit.tabId}
                            type="button"
                            className={`bw-rrow${
                                lead && group.key === props.groups[0].key && hit.tabId === lead.tabId
                                    ? ' is-first'
                                    : ''
                            }`}
                            onClick={() => props.onPick(group, hit.tabId)}
                            disabled={props.disabled}
                            title={`${hit.title || hit.url}\n${hit.url}`}
                        >
                            <span className="bw-fav">{initialOf(hit.title, hit.url)}</span>
                            <span className="bw-rcopy">
                                <span className="bw-rtitle">
                                    <Marked text={hit.title || hit.url} tokens={tokens} />
                                </span>
                                <span className="bw-rurl">
                                    <Marked text={hit.url} tokens={tokens} />
                                </span>
                            </span>
                        </button>
                    ))}
                </section>
            ))}
        </>
    );
}
