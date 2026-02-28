import { WebServiceUI } from './WebServiceUI';

export function CursorWebUI() {
    return (
        <WebServiceUI
            port={3001}
            title="Cursor Web"
            statusEndpoint="/api/cursor-web/status"
            startEndpoint="/api/cursor-web/start"
            stopEndpoint="/api/cursor-web/stop"
            statusStreamEndpoint="/api/cursor-web/status-stream"
            startStreamEndpoint="/api/cursor-web/start-stream"
            stopStreamEndpoint="/api/cursor-web/stop-stream"
            installCommand="npm install -g @siteboon/claude-code-ui"
            authHint="Make sure Cursor CLI is installed and authenticated."
            startCommandPrefix="npx @siteboon/claude-code-ui --port"
            backPath="../experimental"
            enableFocusMode={true}
        />
    );
}
