import { useState, useCallback, useRef } from 'react';
import { Outlet } from 'react-router-dom';
import type { TunnelProvider } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { useTabNavigate } from '../../hooks/useTabNavigate';
import { NavTabs } from './types';
import { useV2Context } from '../V2Context';
import { fetchRandomDomain } from '../../api/domains';
import type { AddPortRequest } from '../../api/ports';

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

    // Refs to track latest state values (avoid stale closures)
    const subdomainRef = useRef(newPortSubdomain);
    const providerRef = useRef(newPortProvider);
    
    // Update refs when state changes
    subdomainRef.current = newPortSubdomain;
    providerRef.current = newPortProvider;

    const generateSubdomain = useCallback(async (baseDomain?: string) => {
        try {
            // Use passed domain parameter, or fall back to current state
            const effectiveDomain = baseDomain || newPortBaseDomain || undefined;
            const domain = await fetchRandomDomain(effectiveDomain);
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

        const isCloudflareProvider = newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned;
        const hasSubdomain = !!newPortSubdomain;
        const hasBaseDomain = !!newPortBaseDomain;

        // Build the request with separate fields for Cloudflare providers
        const req: AddPortRequest = {
            port: portNum,
            label: newPortLabel || `Port ${portNum}`,
            provider: newPortProvider,
        };

        // For Cloudflare providers, pass subdomain and base_domain separately
        if (isCloudflareProvider) {
            if (hasSubdomain) {
                req.subdomain = newPortSubdomain;
            }
            if (hasBaseDomain) {
                req.baseDomain = newPortBaseDomain;
            }
            // If both are provided, construct the full label
            if (hasSubdomain && hasBaseDomain) {
                req.label = `${newPortSubdomain}.${newPortBaseDomain}`;
            }
        }

        console.log('[PortsLayout] Adding port:', {
            portNum,
            isCloudflareProvider,
            hasSubdomain,
            hasBaseDomain,
            newPortSubdomain,
            newPortBaseDomain,
            newPortLabel,
            req
        });

        try {
            setPortActionError(null);
            await addPort(req);
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
            // Generate subdomain when opening form with Cloudflare provider
            // Use ref to get latest provider state (avoid stale closure)
            const currentProvider = providerRef.current;
            if (newValue && (currentProvider === TunnelProviders.CloudflareTunnel || currentProvider === TunnelProviders.CloudflareOwned)) {
                generateSubdomain(newPortBaseDomain || undefined);
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
            // Use ref to get latest subdomain state (avoid stale closure)
            const currentProvider = providerRef.current;
            const currentSubdomain = subdomainRef.current;
            if (domain && !currentSubdomain && (currentProvider === TunnelProviders.CloudflareTunnel || currentProvider === TunnelProviders.CloudflareOwned)) {
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
