import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import type { TunnelProvider } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { useTabNavigate } from '../../hooks/useTabNavigate';
import { NavTabs } from './types';
import { useV2Context } from '../V2Context';

export interface PortsOutletContext {
    ports: ReturnType<typeof useV2Context>['portForwards']['ports'];
    availableProviders: ReturnType<typeof useV2Context>['portForwards']['providers'];
    loading: boolean;
    error: string | null;
    actionError: string | null;
    showNewForm: boolean;
    onToggleNewForm: () => void;
    newPortNumber: string;
    newPortLabel: string;
    newPortProvider: TunnelProvider;
    onPortNumberChange: (value: string) => void;
    onPortLabelChange: (value: string) => void;
    onPortProviderChange: (value: TunnelProvider) => void;
    onAddPort: () => void;
    onRemovePort: (port: number) => void;
    navigateToView: (view: string) => void;
}

export function PortsLayout() {
    const {
        portForwards: { ports, providers: availableProviders, loading, error, addPort, removePort },
    } = useV2Context();
    const navigateToView = useTabNavigate(NavTabs.Ports);

    const [showNewPortForm, setShowNewPortForm] = useState(false);
    const [newPortNumber, setNewPortNumber] = useState('');
    const [newPortLabel, setNewPortLabel] = useState('');
    const [newPortProvider, setNewPortProvider] = useState<TunnelProvider>(TunnelProviders.Localtunnel);
    const [portActionError, setPortActionError] = useState<string | null>(null);

    const handleAddPort = async () => {
        const portNum = parseInt(newPortNumber, 10);
        if (!portNum || portNum <= 0 || portNum > 65535) return;

        const label = newPortLabel || `Port ${portNum}`;
        const provider = newPortProvider;

        try {
            setPortActionError(null);
            await addPort(portNum, label, provider);
            setShowNewPortForm(false);
            setNewPortNumber('');
            setNewPortLabel('');
        } catch (err) {
            setPortActionError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleRemovePort = async (port: number) => {
        try {
            setPortActionError(null);
            await removePort(port);
        } catch (err) {
            setPortActionError(err instanceof Error ? err.message : String(err));
        }
    };

    const ctx: PortsOutletContext = {
        ports,
        availableProviders,
        loading,
        error,
        actionError: portActionError,
        showNewForm: showNewPortForm,
        onToggleNewForm: () => setShowNewPortForm(!showNewPortForm),
        newPortNumber,
        newPortLabel,
        newPortProvider,
        onPortNumberChange: setNewPortNumber,
        onPortLabelChange: setNewPortLabel,
        onPortProviderChange: setNewPortProvider,
        onAddPort: handleAddPort,
        onRemovePort: handleRemovePort,
        navigateToView,
    };

    return <Outlet context={ctx} />;
}
