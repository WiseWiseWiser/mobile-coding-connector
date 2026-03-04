import { useNavigate } from 'react-router-dom';
import { useCurrent } from './useCurrent';
import { useV2Context } from '../v2/V2Context';
import type { NavTab } from '../v2/mcc/types';
import { projectTabPath } from '../route/route';

/**
 * Returns a navigate function scoped to a specific tab.
 * Handles the project/non-project URL prefix automatically.
 *
 * Usage:
 *   const navigateToView = useTabNavigate(NavTabs.Agent);
 *   navigateToView('opencode');        // → /project/{name}/agent/opencode
 *   navigateToView('');                // → /project/{name}/agent
 */
export function useTabNavigate(tab: NavTab) {
    const navigate = useNavigate();
    const { currentProject } = useV2Context();
    const currentProjectRef = useCurrent(currentProject);

    return (view?: string) => {
        const proj = currentProjectRef.current;
        navigate(projectTabPath(proj?.name, tab, view), { replace: true });
    };
}
