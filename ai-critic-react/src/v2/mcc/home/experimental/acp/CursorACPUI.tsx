import { ACPSessionList } from './ACPSessionList';

export function CursorACPUI() {
    return (
        <ACPSessionList
            title="Cursor UI (ACP)"
            agentName="Cursor"
            apiPrefix="/api/agent/acp/cursor"
            backPath="../experimental"
            settingsPath="../acp/cursor/settings"
        />
    );
}
