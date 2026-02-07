import { useOutletContext } from 'react-router-dom';
import type { PortsOutletContext } from './PortsLayout';
import { CloudflareDiagnosticsView } from './PortForwardingView';

export function CloudflareDiagnosticsRoute() {
    const ctx = useOutletContext<PortsOutletContext>();

    return <CloudflareDiagnosticsView onBack={() => ctx.navigateToView('')} />;
}
