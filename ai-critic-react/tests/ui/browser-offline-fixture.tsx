import { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserOffline } from '../../src/v2/mcc/browser/BrowserOffline';
import { useBrowserStartup } from '../../src/v2/mcc/browser/useBrowserStartup';
import '../../src/v2/mcc/browser/browser.css';

// Standalone Vite fixture, not an HMR component module.
// eslint-disable-next-line react-refresh/only-export-components
function Fixture() {
    const [refreshes, setRefreshes] = useState(0);
    const startup = useBrowserStartup(async () => { setRefreshes((n) => n + 1); });
    return <div className="bw-stage" style={{ width: '100vw', height: '100vh' }} data-refreshes={refreshes}><BrowserOffline busy={startup.busy} error={startup.error} onStart={() => void startup.start()} /></div>;
}
createRoot(document.getElementById('root')!).render(<Fixture />);
