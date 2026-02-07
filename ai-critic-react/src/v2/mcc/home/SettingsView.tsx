import { useNavigate } from 'react-router-dom';
import { SecuritySection } from './settings/SecuritySection';
import { WebAccessSection } from './settings/WebAccessSection';
import { GitSettingsContent } from './settings/GitSettings';
import { CloudflareSettingsContent } from './settings/CloudflareSettingsView';
import { TerminalSection } from './settings/TerminalSection';
import './DiagnoseView.css'; // Shared styles: .diagnose-view, .diagnose-section, .diagnose-section-title, .diagnose-loading, .diagnose-error
import './settings/GitSettings.css';
import './settings/CloudflareSettingsView.css';
import './settings/TerminalSection.css';
import './settings/SettingsView.css';

export function SettingsView() {
    const navigate = useNavigate();

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Settings</h2>
            </div>

            <WebAccessSection />

            <div className="diagnose-section">
                <h3 className="diagnose-section-title">Git</h3>
                <GitSettingsContent />
            </div>

            <SecuritySection />

            <TerminalSection />

            <div className="diagnose-section">
                <h3 className="diagnose-section-title">Cloudflare</h3>
                <CloudflareSettingsContent />
            </div>

            <div className="settings-export-import">
                <button className="settings-export-import-btn" onClick={() => navigate('export')}>
                    Export
                </button>
                <button className="settings-export-import-btn" onClick={() => navigate('import')}>
                    Import
                </button>
            </div>
        </div>
    );
}
