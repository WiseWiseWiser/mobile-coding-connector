import { WebServiceUI } from './WebServiceUI';

export function OpencodeWebUI() {
    return (
        <WebServiceUI
            port={4096}
            title="OpenCode Web"
            statusEndpoint="/api/agents/opencode/web-status"
            startEndpoint="/api/agents/opencode/exposed-server/start"
            stopEndpoint="/api/agents/opencode/exposed-server/stop"
            startStreamEndpoint="/api/agents/opencode/exposed-server/start/stream"
            stopStreamEndpoint="/api/agents/opencode/exposed-server/stop/stream"
            installCommand="npm install -g opencode"
            authHint="Make sure OpenCode CLI is installed and authenticated."
            startCommandPrefix="opencode web --port"
            backPath="../experimental"
            enableFocusMode={true}
        />
    );
}
