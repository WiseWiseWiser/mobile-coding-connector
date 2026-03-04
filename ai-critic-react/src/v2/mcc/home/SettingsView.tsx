import '../theme.css';
import { useNavigate } from 'react-router-dom';
import { SecuritySection } from './settings/SecuritySection';
import { WebAccessSection } from './settings/WebAccessSection';
import { ExposedUrlsSection } from './settings/ExposedUrlsSection';
import { GitSettingsContent } from './settings/GitSettings';
import { CloudflareSettingsContent } from './settings/CloudflareSettingsView';
import { TerminalSection } from './settings/TerminalSection';
import { ServerSettingsSection } from './settings/ServerSettingsSection';
import { AIModelsSection } from './settings/AIModelsSection';
import { ProxySettingsSection } from './settings/ProxySettingsSection';
import { ExportButton } from '../../../pure-view/buttons/ExportButton';
import { ImportButton } from '../../../pure-view/buttons/ImportButton';
import { PageView } from '../../../pure-view/PageView';
import { Section } from '../../../pure-view/Section';
import './settings/GitSettings.css';
import './settings/CloudflareSettingsView.css';
import './settings/TerminalSection.css';
import './settings/ExposedUrlsSection.css';
import './settings/SettingsView.css';
import './settings/ServerSettingsSection.css';
import './settings/AIModelsSection.css';
import './settings/LogsView.css';

export function SettingsView() {
    const navigate = useNavigate();

    return (
        <PageView>
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Settings</h2>
            </div>

            <WebAccessSection />

            <ExposedUrlsSection />

            <Section title="Git">
                <GitSettingsContent />
            </Section>

            <SecuritySection />

            <TerminalSection />

            <ProxySettingsSection />

            <AIModelsSection />

            <Section title="Cloudflare">
                <CloudflareSettingsContent />
            </Section>

            <Section title="Server">
                <ServerSettingsSection />
            </Section>

            <div className="settings-export-import">
                <ImportButton onClick={() => navigate('import')} />
                <ExportButton onClick={() => navigate('export')} />
            </div>
        </PageView>
    );
}
