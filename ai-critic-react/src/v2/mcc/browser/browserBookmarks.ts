import { bookmarkFallbackName, isHttpURL } from './browserAddress.ts';

export type BookmarkSource = 'chrome' | 'local';

/** A bookmark tree from the app-local store or a read-only Chrome profile. */
export interface BookmarkDocument {
    version: number;
    roots: BookmarkNode[];
    source?: string;
    profile?: string;
}

export interface BookmarkNode {
    type: string;
    id: string;
    name: string;
    url?: string;
    browser?: string | null;
    children?: BookmarkNode[];
}

export interface BookmarkItem {
    id: string;
    name: string;
    url: string;
    /** Folder path inside the source tree, without the root label. */
    path: string;
}

/** A navigable folder; `path` includes the root label for search results. */
export interface BookmarkFolder {
    id: string;
    name: string;
    path: string;
    folders: BookmarkFolder[];
    links: BookmarkItem[];
    /** Recursive number of links anywhere below this folder. */
    count: number;
}

export interface BookmarkTree {
    roots: BookmarkFolder[];
    total: number;
}

/** The palette is a quick launcher, not a bookmark manager. */
export const MAX_VISIBLE_BOOKMARKS = 120;

/** Depth-first flatten; folders contribute their name to the path. */
export function flattenBookmarks(doc: BookmarkDocument | null | undefined): BookmarkItem[] {
    if (!doc || !Array.isArray(doc.roots)) return [];
    const items: BookmarkItem[] = [];
    const walk = (nodes: BookmarkNode[] | undefined, path: string[]) => {
        for (const node of nodes ?? []) {
            if (!node || typeof node !== 'object') continue;
            if (node.type === 'folder') {
                walk(node.children, [...path, node.name]);
                continue;
            }
            if (node.type !== 'url' || !isHttpURL(node.url)) continue;
            const url = node.url as string;
            items.push({
                id: node.id || url,
                name: node.name?.trim() || bookmarkFallbackName(url),
                url,
                path: path.filter(Boolean).join(' / '),
            });
        }
    };
    walk(doc.roots, []);
    return items;
}

/** Case-insensitive search over title, folder path and URL. */
export function matchBookmarks(items: BookmarkItem[], query: string): BookmarkItem[] {
    const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length) return items;
    return items.filter((item) => {
        const haystack = `${item.name}\n${item.path}\n${item.url}`.toLowerCase();
        return terms.every((term) => haystack.includes(term));
    });
}

export function filterBookmarks(items: BookmarkItem[], query: string, limit = MAX_VISIBLE_BOOKMARKS): BookmarkItem[] {
    return matchBookmarks(items, query).slice(0, limit);
}

/** Build the folder tree once; folders keep their source order. */
export function buildBookmarkTree(doc: BookmarkDocument | null | undefined): BookmarkTree {
    const roots = (doc?.roots ?? [])
        .filter((node) => node && typeof node === 'object')
        .map((node, index) => toFolder(node, [], `root-${index}`));
    return { roots, total: roots.reduce((sum, root) => sum + root.count, 0) };
}

function toFolder(node: BookmarkNode, parentPath: string[], fallbackId: string): BookmarkFolder {
    const name = node.name?.trim() || 'Untitled folder';
    const path = [...parentPath, name];
    const folders: BookmarkFolder[] = [];
    const links: BookmarkItem[] = [];
    let count = 0;

    // Chrome roots are always folders, but a local document may put a page at
    // the top level; keep it launchable instead of dropping it from the tree.
    if (node.type === 'url' && isHttpURL(node.url)) {
        const url = node.url as string;
        links.push({
            id: node.id || url,
            name: name === 'Untitled folder' ? bookmarkFallbackName(url) : name,
            url,
            path: path.join(' / '),
        });
        return {
            id: node.id || path.join('/') || fallbackId,
            name,
            path: path.join(' / '),
            folders,
            links,
            count: 1,
        };
    }

    for (const [index, child] of (node.children ?? []).entries()) {
        if (!child || typeof child !== 'object') continue;
        if (child.type === 'folder') {
            const folder = toFolder(child, path, `${fallbackId}-${index}`);
            folders.push(folder);
            count += folder.count;
            continue;
        }
        if (child.type !== 'url' || !isHttpURL(child.url)) continue;
        const url = child.url as string;
        links.push({
            id: child.id || url,
            name: child.name?.trim() || bookmarkFallbackName(url),
            url,
            path: path.join(' / '),
        });
        count += 1;
    }

    return {
        id: node.id || path.join('/') || fallbackId,
        name,
        path: path.join(' / '),
        folders,
        links,
        count,
    };
}

/** Every launchable link, in tree order. Search stays global to the source. */
export function flattenTreeLinks(tree: BookmarkTree): BookmarkItem[] {
    const items: BookmarkItem[] = [];
    const walk = (folders: BookmarkFolder[]) => {
        for (const folder of folders) {
            items.push(...folder.links);
            walk(folder.folders);
        }
    };
    walk(tree.roots);
    return items;
}

export function searchBookmarkTree(tree: BookmarkTree, query: string): BookmarkItem[] {
    return matchBookmarks(flattenTreeLinks(tree), query);
}

/** Root plus each navigated folder; an unknown id truncates the trail. */
export function bookmarkFolderTrail(root: BookmarkFolder, ids: string[]): BookmarkFolder[] {
    const trail = [root];
    for (const id of ids) {
        const next = trail[trail.length - 1].folders.find((folder) => folder.id === id);
        if (!next) break;
        trail.push(next);
    }
    return trail;
}

/** A URL is already saved when the exact page URL is in the tree. */
export function hasBookmark(items: BookmarkItem[], url: string | null | undefined): boolean {
    if (!url) return false;
    const trimmed = url.trim();
    return items.some((item) => item.url === trimmed);
}

/** Entries at the root have no folder path; give the empty state a name. */
export function bookmarkPath(item: BookmarkItem): string {
    return item.path || 'Bookmarks';
}

/** Header kicker shared by the palette and phone sheet. */
export function bookmarkSourceKicker(source: BookmarkSource): string {
    return source === 'chrome' ? 'BOOKMARKS · CHROME' : 'LOCAL BOOKMARKS';
}

/** One line telling the reader which profile this list came from. */
export function bookmarkSourceNote(source: BookmarkSource, profile: string): string {
    if (source === 'local') return 'On this Mac · editable';
    const name = profile.trim();
    return `${name || 'Default profile'} · read-only`;
}

export function bookmarkCountLabel(count: number): string {
    return `${count} saved`;
}

/** Always offered when Chrome is empty, so a fresh profile can still save. */
export function localBookmarksLink(count: number | null): string {
    return count !== null && count > 0 ? `Local bookmarks (${count}) →` : 'Local bookmarks →';
}

/** Footer ownership line for the palette and sheet. */
export function bookmarkSourceFooter(source: BookmarkSource): string {
    return source === 'chrome' ? 'Chrome · read-only' : '~/.ai-critic/bookmarks.json';
}
