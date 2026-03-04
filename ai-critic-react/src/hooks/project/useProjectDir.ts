import { useState, useEffect } from 'react';
import { resolveProjectDir } from '../../api/projects';
import { useWorktreeRoute } from './useWorktreeRoute';

const dirCache = new Map<string, string>();

function cacheKey(name: string, worktreeId: number): string {
    if (worktreeId === 0) return name;
    return `${name}~${worktreeId}`;
}

export function useProjectDir(): { projectDir: string; projectDirLoading: boolean } {
    const { projectName, worktreeId } = useWorktreeRoute();
    const key = projectName ? cacheKey(projectName, worktreeId) : '';
    const cached = key ? dirCache.get(key) : undefined;

    const [dir, setDir] = useState(cached || '');
    const [loading, setLoading] = useState(!cached && !!projectName);

    useEffect(() => {
        if (!projectName) return;

        const k = cacheKey(projectName, worktreeId);
        const hit = dirCache.get(k);
        if (hit) {
            setDir(hit);
            setLoading(false);
            return;
        }

        setLoading(true);
        const wtId = worktreeId > 0 ? String(worktreeId) : undefined;
        resolveProjectDir(projectName, wtId)
            .then(resolved => {
                dirCache.set(k, resolved);
                setDir(resolved);
            })
            .catch(() => {
                setDir('');
            })
            .finally(() => {
                setLoading(false);
            });
    }, [projectName, worktreeId]);

    return { projectDir: dir, projectDirLoading: loading };
}
