import { useRef } from 'react';
import { useLocation, Outlet, useOutletContext } from 'react-router-dom';
import type { FilesOutletContext } from './FilesLayout';
import './FilesView.css';

const FilesSubTabs = {
    Checkpoints: 'checkpoints',
    Browse: 'browse',
} as const;

type FilesSubTab = typeof FilesSubTabs[keyof typeof FilesSubTabs];

export function FilesTabLayout() {
    const ctx = useOutletContext<FilesOutletContext>();
    const location = useLocation();

    // Determine active sub-tab from URL
    const isBrowseView = location.pathname.includes('/browse');
    const activeSubTab: FilesSubTab = isBrowseView ? FilesSubTabs.Browse : FilesSubTabs.Checkpoints;

    // Remember the last browse path so we can restore it when switching tabs
    const lastBrowsePathRef = useRef('browse');
    if (isBrowseView) {
        const browseMatch = location.pathname.match(/\/files\/browse(\/.*)?$/);
        if (browseMatch) {
            lastBrowsePathRef.current = 'browse' + (browseMatch[1] || '');
        }
    }

    return (
        <div className="mcc-files">
            {/* Sub-tab bar */}
            <div className="mcc-files-subtabs">
                <button
                    className={`mcc-files-subtab${activeSubTab === FilesSubTabs.Checkpoints ? ' mcc-files-subtab-active' : ''}`}
                    onClick={() => ctx.navigateToView()}
                >
                    Checkpoints
                </button>
                <button
                    className={`mcc-files-subtab${activeSubTab === FilesSubTabs.Browse ? ' mcc-files-subtab-active' : ''}`}
                    onClick={() => ctx.navigateToView(lastBrowsePathRef.current)}
                >
                    Browse Files
                </button>
            </div>

            <Outlet context={ctx} />
        </div>
    );
}
