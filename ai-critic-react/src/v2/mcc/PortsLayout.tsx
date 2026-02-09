import { useState, useCallback } from 'react';
import { Outlet } from 'react-router-dom';
import type { TunnelProvider } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { useTabNavigate } from '../../hooks/useTabNavigate';
import { NavTabs } from './types';
import { useV2Context } from '../V2Context';
import { fetchRandomDomain } from '../../api/domains';

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
    newPortSubdomain: string;
    newPortBaseDomain: string;
    onPortNumberChange: (value: string) => void;
    onPortLabelChange: (value: string) => void;
    onPortProviderChange: (value: TunnelProvider) => void;
    onPortSubdomainChange: (value: string) => void;
    onPortBaseDomainChange: (value: string) => void;
    onGenerateSubdomain: () => void;
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
    const [newPortSubdomain, setNewPortSubdomain] = useState('');
    const [newPortBaseDomain, setNewPortBaseDomain] = useState('');
    const [portActionError, setPortActionError] = useState<string | null>(null);

    const generateSubdomain = useCallback(async (baseDomain?: string) => {
        try {
            const domain = await fetchRandomDomain(baseDomain || newPortBaseDomain || undefined);
            // Extract just the subdomain part (before the first dot)
            const subdomain = domain.split('.')[0];
            setNewPortSubdomain(subdomain);
        } catch {
            // Fallback: generate a simple random string
            const random = Math.random().toString(36).substring(2, 8);
            setNewPortSubdomain(random);
        }
    }, [newPortBaseDomain]);

    const handleAddPort = async () => {
        const portNum = parseInt(newPortNumber, 10);
        if (!portNum || portNum <= 0 || portNum > 65535) return;

        // For Cloudflare providers, combine subdomain + base domain as label
        let label = newPortLabel || `Port ${portNum}`;
        if ((newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned) && newPortSubdomain && newPortBaseDomain) {
            label = `${newPortSubdomain}.${newPortBaseDomain}`;
        }
        const provider = newPortProvider;

        try {
            setPortActionError(null);
            await addPort(portNum, label, provider);
            setShowNewPortForm(false);
            setNewPortNumber('');
            setNewPortLabel('');
            setNewPortSubdomain('');
            setNewPortBaseDomain('');
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
        onToggleNewForm: () => {
            const newValue = !showNewPortForm;
            setShowNewPortForm(newValue);
            // Generate subdomain when opening form with Cloudflare provider and base domain already selected
            if (newValue && (newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned)) {
                if (newPortBaseDomain && !newPortSubdomain) {
                    generateSubdomain(newPortBaseDomain);
                }
            }
        },
        newPortNumber,
        newPortLabel,
        newPortProvider,
        newPortSubdomain,
        newPortBaseDomain,
        onPortNumberChange: setNewPortNumber,
        onPortLabelChange: setNewPortLabel,
        onPortProviderChange: (provider: TunnelProvider) => {
            setNewPortProvider(provider);
            // Reset subdomain-related state when switching providers
            if (provider !== TunnelProviders.CloudflareTunnel && provider !== TunnelProviders.CloudflareOwned) {
                setNewPortSubdomain('');
                setNewPortBaseDomain('');
            }
        },
        onPortSubdomainChange: setNewPortSubdomain,
        onPortBaseDomainChange: (domain: string) => {
            setNewPortBaseDomain(domain);
            // Generate subdomain when selecting a base domain for Cloudflare providers
            if (domain && (newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned)) {
                generateSubdomain(domain);
            }
        },
        onGenerateSubdomain: () => generateSubdomain(),
        onAddPort: handleAddPort,
        onRemovePort: handleRemovePort,
        navigateToView,
    };

    return <Outlet context={ctx} />;
}
