import { useNavigate, useLocation } from 'react-router-dom';
import { useV2Context } from '../v2/V2Context';
import { useCurrent } from './useCurrent';
import type { NavTab } from '../v2/mcc/types';

interface UseTabHistoryOptions {
    /**
     * Override the default back path when there's no history.
     * If not provided, defaults to the tab's root path.
     */
    defaultBackPath?: string;
}

/**
 * Per-tab navigation history management.
 * 
 * This hook provides:
 * - pushView: Navigate to a new view within the current tab (adds to history)
 * - goBack: Go back within the current tab's history (or to tab root if at start)
 * - replaceView: Replace current view without adding to history
 * 
 * The history is stored in V2Context so it persists across component remounts.
 */
export function useTabHistory(tab: NavTab, options?: UseTabHistoryOptions) {
    const navigate = useNavigate();
    const location = useLocation();
    const { currentProject, tabHistories, pushTabHistory, popTabHistory } = useV2Context();

    // Use refs to access latest values without adding dependencies
    const currentProjectRef = useCurrent(currentProject);
    const locationRef = useCurrent(location);
    const tabHistoriesRef = useCurrent(tabHistories);
    const optionsRef = useCurrent(options);

    // Build the base path for this tab
    const getTabBase = () => {
        const proj = currentProjectRef.current;
        return proj
            ? `/project/${encodeURIComponent(proj.name)}/${tab}`
            : `/${tab}`;
    };

    // Get the default back path (either from options or tab base)
    const getDefaultBackPath = () => {
        return optionsRef.current?.defaultBackPath ?? getTabBase();
    };

    // Push a new view onto the tab's history stack
    const pushView = (view: string) => {
        const tabBase = getTabBase();
        const fullPath = view ? `${tabBase}/${view}` : tabBase;
        
        // Record current path in history before navigating
        pushTabHistory(tab, locationRef.current.pathname);
        navigate(fullPath);
    };

    // Replace current view without adding to history
    const replaceView = (view?: string) => {
        const tabBase = getTabBase();
        const fullPath = view ? `${tabBase}/${view}` : tabBase;
        navigate(fullPath, { replace: true });
    };

    // Go back within the tab's history
    const goBack = () => {
        const history = tabHistoriesRef.current[tab] || [];
        
        if (history.length > 0) {
            // Pop the last entry and navigate to it
            const previousPath = popTabHistory(tab);
            if (previousPath) {
                navigate(previousPath, { replace: true });
                return;
            }
        }
        
        // No history - go to default back path (tab root or custom path)
        const backPath = getDefaultBackPath();
        navigate(backPath, { replace: true });
    };

    // Check if we can go back (have history)
    const canGoBack = (tabHistories[tab]?.length || 0) > 0;

    return {
        pushView,
        replaceView,
        goBack,
        canGoBack,
        currentHistory: tabHistories[tab] || [],
    };
}
