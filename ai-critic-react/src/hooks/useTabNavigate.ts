import { useNavigate } from 'react-router-dom';
import { useCurrent } from './useCurrent';
import { useWorktreeRoute } from './project/useWorktreeRoute';
import type { NavTab } from '../v2/mcc/types';
import { buildProjectNavPath } from '../route/route';

/**
 * Returns a navigate function scoped to a specific tab.
 * Handles the project/non-project URL prefix automatically,
 * preserving the worktree suffix (e.g. "proj~2") from the URL.
 *
 * Usage:
 *   const navigateToView = useTabNavigate(NavTabs.Agent);
 *   navigateToView('opencode');        // → /project/{name~wt}/agent/opencode
 *   navigateToView('');                // → /project/{name~wt}/agent
 */
export function useTabNavigate(tab: NavTab) {
    const navigate = useNavigate();
    const { fullProjectName } = useWorktreeRoute();
    const fullProjectNameRef = useCurrent(fullProjectName);

    return (view?: string) => {
        navigate(buildProjectNavPath(fullProjectNameRef.current || undefined, tab, view), { replace: true });
    };
}
