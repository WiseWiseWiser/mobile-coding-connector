import { useState } from 'react';
import { FileBrowser } from '../../../components/chooser/FileBrowser';
import type { SelectMode } from '../../../components/chooser/FileBrowser';
import { SelectModes } from '../../../components/chooser/FileBrowser';
import '../../../components/chooser/FileBrowser.css';

interface ServerFileBrowserProps {
    selectMode?: SelectMode;
    onSelect?: (path: string | null) => void;
    onDirectoryChange?: (path: string) => void;
    initialDir?: string;
}

export function ServerFileBrowser({
    selectMode = SelectModes.File,
    onSelect,
    onDirectoryChange,
    initialDir = '/',
}: ServerFileBrowserProps) {
    const [pathInput, setPathInput] = useState(initialDir);
    const [browseDir, setBrowseDir] = useState(initialDir);
    const [key, setKey] = useState(0);

    const handlePathGo = () => {
        const trimmed = pathInput.trim();
        if (!trimmed) return;
        setBrowseDir(trimmed);
        setKey(k => k + 1);
    };

    const handlePathKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handlePathGo();
        }
    };

    return (
        <>
            <div className="fb-section">
                <label className="fb-label">Server Path</label>
                <div className="fb-path-row">
                    <input
                        type="text"
                        className="fb-path-input"
                        placeholder="/path/to/directory"
                        value={pathInput}
                        onChange={e => setPathInput(e.target.value)}
                        onKeyDown={handlePathKeyDown}
                    />
                    <button
                        className="fb-go-btn"
                        onClick={handlePathGo}
                        disabled={!pathInput.trim()}
                    >
                        Go
                    </button>
                </div>
            </div>
            <FileBrowser
                key={key}
                selectMode={selectMode}
                onSelect={onSelect}
                onDirectoryChange={(path) => {
                    setPathInput(path);
                    onDirectoryChange?.(path);
                }}
                initialDir={browseDir}
            />
        </>
    );
}

export { SelectModes };
export type { SelectMode };
