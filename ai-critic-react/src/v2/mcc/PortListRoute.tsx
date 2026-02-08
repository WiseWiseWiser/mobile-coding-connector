import { useOutletContext } from 'react-router-dom';
import type { PortsOutletContext } from './PortsLayout';
import { PortForwardingView } from './PortForwardingView';
import { useLocalPorts } from '../../hooks/useLocalPorts';

export function PortListRoute() {
    const ctx = useOutletContext<PortsOutletContext>();
    const { ports: localPorts, loading: localPortsLoading, error: localPortsError } = useLocalPorts();

    const handleForwardLocalPort = (port: number) => {
        ctx.onPortNumberChange(port.toString());
        ctx.onPortLabelChange(`Port ${port}`);
        if (!ctx.showNewForm) {
            ctx.onToggleNewForm();
        }
    };

    return (
        <PortForwardingView
            ports={ctx.ports}
            availableProviders={ctx.availableProviders}
            loading={ctx.loading}
            error={ctx.error}
            actionError={ctx.actionError}
            showNewForm={ctx.showNewForm}
            onToggleNewForm={ctx.onToggleNewForm}
            newPortNumber={ctx.newPortNumber}
            newPortLabel={ctx.newPortLabel}
            newPortProvider={ctx.newPortProvider}
            onPortNumberChange={ctx.onPortNumberChange}
            onPortLabelChange={ctx.onPortLabelChange}
            onPortProviderChange={ctx.onPortProviderChange}
            onAddPort={ctx.onAddPort}
            onRemovePort={ctx.onRemovePort}
            onNavigateToView={ctx.navigateToView}
            localPorts={localPorts}
            localPortsLoading={localPortsLoading}
            localPortsError={localPortsError}
            onForwardLocalPort={handleForwardLocalPort}
        />
    );
}
