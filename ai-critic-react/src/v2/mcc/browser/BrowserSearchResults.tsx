import { matchCount, searchTokens } from './tabSearch';
import type { SearchGroup } from './tabSearch';
import { Marked } from './Marked';

interface BrowserSearchResultsProps {
    query: string;
    groups: SearchGroup[];
    /** Windows searched, so an empty answer can say how wide it looked. */
    windowCount: number;
    /** Jump to a tab, switching windows when the hit belongs to another one. */
    onPick: (group: SearchGroup, tabId: number) => void;
    disabled?: boolean;
}

/**
 * In-place results: they take the panes' place under the search row, grouped by
 * window in the rail's own order, with the matched text highlighted on both the
 * title and the URL.
 */
export function BrowserSearchResults(props: BrowserSearchResultsProps) {
    const tokens = searchTokens(props.query);
    const count = matchCount(props.groups);
    const windowWord = props.groups.length === 1 ? 'window' : 'windows';
    const tabWord = count === 1 ? 'tab' : 'tabs';
    return (
        <div className="bwp-sresults">
            <div className="bwp-smeta">
                <span className="bwp-scount">
                    {count} {tabWord} in {props.groups.length} {windowWord}
                </span>
                <span className="bwp-snote">titles and URLs</span>
            </div>
            {count === 0 ? (
                <div className="bwp-sempty">
                    <span className="bwp-semptyq">“{props.query.trim()}”</span>
                    <span className="bwp-semptyh">
                        No tab title or URL matches in {props.windowCount}{' '}
                        {props.windowCount === 1 ? 'window' : 'windows'}.
                    </span>
                </div>
            ) : null}
            {props.groups.map((group) => (
                <section className="bwp-sgroup" key={group.key}>
                    <div className="bwp-sghead">
                        <span className="bwp-sgname">{group.title}</span>
                        <span className="bwp-sgnum">{group.hits.length}</span>
                    </div>
                    <ul className="bwp-slist">
                        {group.hits.map((hit) => (
                            <li key={hit.tabId}>
                                <button
                                    type="button"
                                    className="bwp-sresult"
                                    onClick={() => props.onPick(group, hit.tabId)}
                                    disabled={props.disabled}
                                    title={`${hit.title || hit.url}\n${hit.url}`}
                                >
                                    <span className="bwp-srtitle">
                                        <Marked text={hit.title || hit.url} tokens={tokens} />
                                    </span>
                                    <span className="bwp-srurl">
                                        <Marked text={hit.url} tokens={tokens} />
                                    </span>
                                </button>
                            </li>
                        ))}
                    </ul>
                </section>
            ))}
        </div>
    );
}
