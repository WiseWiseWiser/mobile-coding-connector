import { useNavigate } from 'react-router-dom';
import { useCurrent } from './useCurrent';
import { useV2Context } from '../v2/V2Context';
import type { NavTab } from '../v2/mcc/types';

/**
 * Returns a navigate function scoped to a specific tab.
 * Handles the project/non-project URL prefix automatically.
 *
 * Usage:
 *   const navigateToView = useTabNavigate(NavTabs.Agent);
 *   navigateToView('opencode');        // → /v2/project/{name}/agent/opencode
 *   navigateToView('');                // → /v2/project/{name}/agent
 */
export function useTabNavigate(tab: NavTab) {
    const navigate = useNavigate();
    const { currentProject } = useV2Context();
    const currentProjectRef = useCurrent(currentProject);

    return (view?: string) => {
        const proj = currentProjectRef.current;
        const base = '/v2';
        const tabBase = proj
            ? `${base}/project/${encodeURIComponent(proj.name)}/${tab}`
            : `${base}/${tab}`;

        navigate(view ? `${tabBase}/${view}` : tabBase, { replace: true });
    };
}
