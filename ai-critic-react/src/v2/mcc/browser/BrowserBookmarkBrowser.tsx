import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { addressLabel, checkAddress, looksLikeAddress } from './browserAddress';
import {
    MAX_VISIBLE_BOOKMARKS,
    bookmarkFolderTrail,
    buildBookmarkTree,
    searchBookmarkTree,
} from './browserBookmarks';
import type { BookmarkDocument, BookmarkFolder, BookmarkItem } from './browserBookmarks';

interface BrowserBookmarkBrowserProps {
    doc: BookmarkDocument | null;
    loading: boolean;
    error: string | null;
    disabled?: boolean;
    /** When supplied, a parent owns the filter and no separate field is shown. */
    filterQuery?: string;
    /** New-tab mode: the field also opens addresses and searches the web. */
    addressMode?: boolean;
    seed?: string;
    autoFocus?: boolean;
    emptyText?: string;
    emptyHint?: string;
    emptyAction?: { label: string; onClick: () => void };
    onOpen: (item: BookmarkItem) => void;
    onCreate?: (value: string) => void;
}

function itemLabel(count: number): string {
    return `${count} ${count === 1 ? 'item' : 'items'}`;
}

/**
 * The one bookmark field: filter the tree while typing, or — on a new tab —
 * treat the same text as an address. Folders open in place with breadcrumbs;
 * a search always scans the whole source and keeps each result's folder path.
 */
export function BrowserBookmarkBrowser({
    doc,
    loading,
    error,
    disabled = false,
    filterQuery,
    addressMode = false,
    seed = '',
    autoFocus = false,
    emptyText,
    emptyHint,
    emptyAction,
    onOpen,
    onCreate,
}: BrowserBookmarkBrowserProps) {
    const [internalQuery, setQuery] = useState(seed);
    const query = filterQuery ?? internalQuery;
    const [fieldError, setFieldError] = useState('');
    const [rootId, setRootId] = useState('');
    const [folderPath, setFolderPath] = useState<string[]>([]);
    const input = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        if (!autoFocus) return;
        const timer = window.setTimeout(() => {
            input.current?.focus();
            input.current?.select();
        }, 40);
        return () => window.clearTimeout(timer);
    }, [autoFocus]);

    const tree = buildBookmarkTree(doc);
    const roots = tree.roots.filter((root) => root.count > 0);
    const root = roots.find((item) => item.id === rootId) ?? roots[0] ?? null;
    const trail = root ? bookmarkFolderTrail(root, folderPath) : [];
    const folder = trail[trail.length - 1] ?? null;
    const trimmed = query.trim();
    const searching = trimmed !== '';
    const addressLike = addressMode && looksLikeAddress(trimmed);
    const matches = searching ? searchBookmarkTree(tree, query) : [];
    const searchRows = matches.slice(0, MAX_VISIBLE_BOOKMARKS);
    const browseRows = folder ? folder.links.slice(0, MAX_VISIBLE_BOOKMARKS) : [];
    const isRoot = trail.length === 1;
    const directCount = folder ? folder.folders.length + folder.links.length : 0;
    const count = isRoot ? directCount : (folder?.count ?? 0);
    const action = !trimmed || addressLike ? 'Open' : 'Search';
    const hint = !addressMode
        ? ''
        : !trimmed
          ? tree.total > 0
              ? `Filter all ${tree.total} links · Enter opens a blank tab`
              : 'Type an address to start'
          : addressLike
            ? 'Enter opens this address in a new tab'
            : `Enter searches the web · ${matches.length} ${matches.length === 1 ? 'match' : 'matches'}`;

    const submit = (event: FormEvent) => {
        event.preventDefault();
        if (!addressMode || disabled || !onCreate) return;
        if (!trimmed) {
            setFieldError('');
            onCreate('');
            return;
        }
        const check = checkAddress(trimmed);
        if (!check.ok) {
            setFieldError(check.error);
            return;
        }
        setFieldError('');
        onCreate(check.target);
    };

    const goRoot = (next: BookmarkFolder) => {
        setRootId(next.id);
        setFolderPath([]);
    };

    const goFolder = (next: BookmarkFolder) => {
        setFolderPath([...trail.slice(1).map((item) => item.id), next.id]);
        setQuery('');
    };

    const goTrail = (depth: number) => {
        setFolderPath(trail.slice(1, depth + 1).map((item) => item.id));
        setQuery('');
    };

    const open = (item: BookmarkItem) => {
        if (disabled) return;
        onOpen(item);
    };

    const empty =
        error !== null ? (
            <div className="bw-bmempty" role="alert">
                <span>Bookmarks could not be read.</span>
                <span className="bw-bmempty-hint">{error}</span>
            </div>
        ) : loading && !tree.roots.length ? (
            <div className="bw-bmempty">Loading bookmarks…</div>
        ) : searching && !matches.length ? (
            <div className="bw-bmempty">
                <span>No bookmarks match “{trimmed}”.</span>
                {addressMode && <span className="bw-bmempty-hint">Press Enter to search the web instead.</span>}
            </div>
        ) : !roots.length ? (
            <div className="bw-bmempty">
                <span>{emptyText ?? 'No bookmarks yet.'}</span>
                {emptyHint && <span className="bw-bmempty-hint">{emptyHint}</span>}
                {emptyAction && (
                    <button
                        type="button"
                        className="bw-bmempty-action"
                        onClick={emptyAction.onClick}
                        disabled={disabled}
                    >
                        {emptyAction.label}
                    </button>
                )}
            </div>
        ) : null;

    return (
        <div className="bw-bm bw-bmbrowser">
            {filterQuery === undefined && <form className={`bw-omni${fieldError ? ' has-error' : ''}`} onSubmit={submit}>
                <span className="bw-omni-icon" aria-hidden="true">
                    {addressLike ? '⌁' : '⌕'}
                </span>
                <input
                    ref={input}
                    className="bw-omni-input"
                    type="text"
                    value={query}
                    onChange={(event) => {
                        setQuery(event.target.value);
                        if (fieldError) setFieldError('');
                    }}
                    placeholder={addressMode ? 'Filter bookmarks or enter an address' : 'Filter bookmarks'}
                    aria-label={addressMode ? 'Filter bookmarks or enter an address' : 'Filter bookmarks'}
                    autoCapitalize="off"
                    autoCorrect="off"
                    spellCheck={false}
                    inputMode="text"
                    enterKeyHint={addressMode ? 'go' : 'search'}
                    disabled={disabled}
                />
                {addressMode && (
                    <button type="submit" className="bw-omni-btn" disabled={disabled}>
                        {action}
                    </button>
                )}
            </form>}
            {filterQuery === undefined && addressMode && <div className="bw-omni-hint">{hint}</div>}
            {fieldError && (
                <div className="bw-omni-error" role="alert">
                    {fieldError}
                </div>
            )}

            <div className="bw-treebar" aria-live="polite">
                {searching ? (
                    <div className="bw-crumbs">
                        <span className="bw-crumb is-current">Search results</span>
                        <span className="bw-scope">all folders</span>
                    </div>
                ) : (
                    <div className="bw-crumbs">
                        {trail.map((item, index) =>
                            index === trail.length - 1 ? (
                                <span key={item.id} className="bw-crumb is-current" aria-current="page">
                                    {item.name}
                                </span>
                            ) : (
                                <span key={item.id} className="bw-crumbstep">
                                    <button type="button" className="bw-crumb" onClick={() => goTrail(index)}>
                                        {item.name}
                                    </button>
                                    <span className="bw-crumbsep" aria-hidden="true">
                                        ›
                                    </span>
                                </span>
                            ),
                        )}
                    </div>
                )}
                <span className="bw-treecount">{searching ? `${matches.length}/${tree.total}` : itemLabel(count)}</span>
            </div>

            {!searching && roots.length > 1 && (
                <div className="bw-roots">
                    {roots.map((item) => (
                        <button
                            key={item.id}
                            type="button"
                            className={`bw-rootpill${item.id === root?.id ? ' is-on' : ''}`}
                            onClick={() => goRoot(item)}
                        >
                            {item.name}
                        </button>
                    ))}
                </div>
            )}

            {empty ?? (
                <div className="bw-bmlist">
                    {searching ? (
                        <>
                            <div className="bw-grouphead">
                                <span>Bookmarks</span>
                                <span>{matches.length}</span>
                            </div>
                            {searchRows.map((item) => (
                                <button
                                    key={`${item.id}:${item.url}`}
                                    type="button"
                                    className="bw-bmrow bw-resultrow"
                                    onClick={() => open(item)}
                                    disabled={disabled}
                                >
                                    <span className="bw-bmtext">
                                        <span className="bw-bmname">{item.name}</span>
                                        <span className="bw-bmurl">{addressLabel(item.url)}</span>
                                    </span>
                                    <span className="bw-bmpath bw-resultpath" title={item.path}>
                                        {item.path}
                                    </span>
                                    <span className="bw-bmgo" aria-hidden="true">
                                        ↗
                                    </span>
                                </button>
                            ))}
                            {matches.length > MAX_VISIBLE_BOOKMARKS && (
                                <div className="bw-cap">
                                    Showing the first {MAX_VISIBLE_BOOKMARKS} of {matches.length}. Type more to narrow
                                    the search.
                                </div>
                            )}
                        </>
                    ) : (
                        <>
                            {folder && folder.folders.length > 0 && (
                                <>
                                    <div className="bw-grouphead">
                                        <span>Folders</span>
                                        <span>{folder.folders.length}</span>
                                    </div>
                                    {folder.folders.map((item) => (
                                        <button
                                            key={item.id}
                                            type="button"
                                            className="bw-bmrow bw-folderrow"
                                            onClick={() => goFolder(item)}
                                        >
                                            <span className="bw-foldericon" aria-hidden="true">
                                                ▸
                                            </span>
                                            <span className="bw-bmtext">
                                                <span className="bw-bmname">{item.name}</span>
                                                <span className="bw-bmurl">Folder</span>
                                            </span>
                                            <span className="bw-foldercount">{item.count}</span>
                                            <span className="bw-bmgo" aria-hidden="true">
                                                ›
                                            </span>
                                        </button>
                                    ))}
                                </>
                            )}
                            {folder && folder.links.length > 0 && (
                                <>
                                    <div className="bw-grouphead">
                                        <span>Bookmarks</span>
                                        <span>{folder.links.length}</span>
                                    </div>
                                    {browseRows.map((item) => (
                                        <button
                                            key={`${item.id}:${item.url}`}
                                            type="button"
                                            className="bw-bmrow"
                                            onClick={() => open(item)}
                                            disabled={disabled}
                                        >
                                            <span className="bw-bmtext">
                                                <span className="bw-bmname">{item.name}</span>
                                                <span className="bw-bmurl">{addressLabel(item.url)}</span>
                                            </span>
                                            <span className="bw-bmgo" aria-hidden="true">
                                                ↗
                                            </span>
                                        </button>
                                    ))}
                                    {folder.links.length > MAX_VISIBLE_BOOKMARKS && (
                                        <div className="bw-cap">
                                            Showing the first {MAX_VISIBLE_BOOKMARKS} of {folder.links.length}. Filter
                                            to find the rest.
                                        </div>
                                    )}
                                </>
                            )}
                            {folder && !folder.folders.length && !folder.links.length && (
                                <div className="bw-bmempty">
                                    <span>This folder is empty.</span>
                                    <span className="bw-bmempty-hint">Type above to search the whole tree.</span>
                                </div>
                            )}
                        </>
                    )}
                </div>
            )}
        </div>
    );
}
