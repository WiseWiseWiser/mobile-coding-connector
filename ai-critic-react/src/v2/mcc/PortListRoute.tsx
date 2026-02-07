import { useOutletContext } from 'react-router-dom';
import type { PortsOutletContext } from './PortsLayout';
import { PortForwardingView } from './PortForwardingView';

export function PortListRoute() {
    const ctx = useOutletContext<PortsOutletContext>();

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
        />
    );
}
