import { useOutletContext, useParams } from 'react-router-dom';
import type { PortsOutletContext } from './PortsLayout';
import { PortDiagnoseView } from './PortForwardingView';

export function PortDiagnoseRoute() {
    const ctx = useOutletContext<PortsOutletContext>();
    const params = useParams<{ port?: string }>();
    const port = parseInt(params.port || '0', 10);
    const portData = ctx.ports.find(p => p.localPort === port);

    return <PortDiagnoseView port={port} portData={portData} onBack={() => ctx.navigateToView('')} />;
}
