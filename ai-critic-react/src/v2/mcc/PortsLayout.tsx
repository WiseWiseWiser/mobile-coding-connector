import { useState, useRef, useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import type { TunnelProvider } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { useTabNavigate } from '../../hooks/useTabNavigate';
import { NavTabs } from './types';
import { useV2Context } from '../V2Context';
import { fetchRandomDomain } from '../../api/domains';
import type { AddPortRequest } from '../../api/ports';
import { fetchAllPortMappingAuth, fetchAllPortMappingNames, type PortMappingAuth, type PortMappingNames } from '../../api/ports';
import type { AuthTokenMode, AuthUserMode } from './ServiceAuthFields';

export interface PortsOutletContext {
    ports: ReturnType<typeof useV2Context>['portForwards']['ports'];
    availableProviders: ReturnType<typeof useV2Context>['portForwards']['providers'];
    loading: boolean;
    error: string | null;
    actionError: string | null;
    showNewForm: boolean;
    onToggleNewForm: () => void;
    newTargetHost: string;
    newTargetScheme: 'http' | 'https';
    onTargetHostChange: (host: string) => void;
    onTargetSchemeChange: (scheme: 'http' | 'https') => void;
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
    requireAuth: boolean;
    authUserMode: AuthUserMode;
    authUser: string;
    authTokenMode: AuthTokenMode;
    authToken: string;
    onRequireAuthChange: (value: boolean) => void;
    onAuthUserModeChange: (value: AuthUserMode) => void;
    onAuthUserChange: (value: string) => void;
    onAuthTokenModeChange: (value: AuthTokenMode) => void;
    onAuthTokenChange: (value: string) => void;
    onAddPort: () => void;
    onRemovePort: (port: number | string) => void;
    navigateToView: (view: string) => void;
}

export function PortsLayout() {
    const {
        portForwards: { ports, providers: availableProviders, loading, error, addPort, removePort },
    } = useV2Context();
    const navigateToView = useTabNavigate(NavTabs.Service);

    const [showNewPortForm, setShowNewPortForm] = useState(false);
    const [newTargetHost, setNewTargetHost] = useState('127.0.0.1');
    const [newTargetScheme, setNewTargetScheme] = useState<'http' | 'https'>('http');
    const [newPortNumber, setNewPortNumber] = useState('');
    const [newPortLabel, setNewPortLabel] = useState('');
    const [newPortProvider, setNewPortProvider] = useState<TunnelProvider>(TunnelProviders.Localtunnel);
    const [newPortSubdomain, setNewPortSubdomain] = useState('');
    const [newPortBaseDomain, setNewPortBaseDomain] = useState('');
    const [portActionError, setPortActionError] = useState<string | null>(null);
    const [portMappingNames, setPortMappingNames] = useState<PortMappingNames>({});
    const [portMappingAuth, setPortMappingAuth] = useState<PortMappingAuth>({});
    const [requireAuth, setRequireAuth] = useState(false);
    const [authUserMode, setAuthUserMode] = useState<AuthUserMode>('any');
    const [authUser, setAuthUser] = useState('');
    const [authTokenMode, setAuthTokenMode] = useState<AuthTokenMode>('shared');
    const [authToken, setAuthToken] = useState('');

    // Refs to track latest state values (avoid stale closures)
    const subdomainRef = useRef(newPortSubdomain);
    const providerRef = useRef(newPortProvider);
    
    // Update refs when state changes
    subdomainRef.current = newPortSubdomain;
    providerRef.current = newPortProvider;

    // Load port mapping names on mount
    useEffect(() => {
        fetchAllPortMappingNames()
            .then(names => setPortMappingNames(names))
            .catch(() => { /* ignore errors */ });
        fetchAllPortMappingAuth()
            .then(auth => setPortMappingAuth(auth))
            .catch(() => { /* ignore errors */ });
    }, []);

    // Apply saved mapping name when port number changes (for Cloudflare providers)
    useEffect(() => {
        if (!newPortNumber || newTargetHost !== '127.0.0.1') return;
        
        const portNum = parseInt(newPortNumber, 10);
        if (!portNum || portNum <= 0) return;

        const savedAuth = portMappingAuth[String(portNum)];
        if (savedAuth?.required) {
            setRequireAuth(true);
            setAuthUserMode(savedAuth.user ? 'fixed' : 'any');
            setAuthUser(savedAuth.user || '');
            setAuthTokenMode(savedAuth.tokenMode === 'custom' ? 'custom' : 'shared');
            setAuthToken(savedAuth.token || '');
        }

        const savedDomain = portMappingNames[String(portNum)];
        if (!savedDomain) return;

        // Only apply if using Cloudflare providers
        const isCloudflareProvider = newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned;
        if (!isCloudflareProvider) return;

        // Parse the saved domain to extract subdomain and base domain
        const parts = savedDomain.split('.');
        if (parts.length >= 2) {
            const subdomain = parts[0];
            const baseDomain = parts.slice(1).join('.');
            
            setNewPortSubdomain(subdomain);
            setNewPortBaseDomain(baseDomain);
        }
    }, [newPortNumber, portMappingNames, portMappingAuth, newPortProvider, newTargetHost]);

    const generateSubdomain = async (baseDomain?: string) => {
        try {
            // Use passed domain parameter, or fall back to current state
            const effectiveDomain = baseDomain || newPortBaseDomain || undefined;
            console.log('[PortsLayout] generateSubdomain called with baseDomain param:', baseDomain, 'effectiveDomain:', effectiveDomain);
            const domain = await fetchRandomDomain(effectiveDomain);
            // Extract just the subdomain part (before the first dot)
            const subdomain = domain.split('.')[0];
            console.log('[PortsLayout] Generated subdomain:', subdomain, 'from domain:', domain);
            setNewPortSubdomain(subdomain);
        } catch (err) {
            console.log('[PortsLayout] generateSubdomain error:', err);
            // Fallback: generate a simple random string
            const random = Math.random().toString(36).substring(2, 8);
            setNewPortSubdomain(random);
        }
    };

    const handleAddPort = async () => {
        const portNum = parseInt(newPortNumber, 10);
        if (!portNum || portNum <= 0 || portNum > 65535) return;

        const isCloudflareProvider = newPortProvider === TunnelProviders.CloudflareTunnel || newPortProvider === TunnelProviders.CloudflareOwned;
        const hasSubdomain = !!newPortSubdomain;
        const hasBaseDomain = !!newPortBaseDomain;

        // Build the request with separate fields for Cloudflare providers
        const req: AddPortRequest = {
            port: portNum,
            auth: { required: requireAuth },
            label: newPortLabel || `Port ${portNum}`,
            provider: newPortProvider,
            target: { host: newTargetHost, scheme: newTargetScheme, port: portNum },
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
        if (requireAuth) {
            req.auth = { required: true };
            if (authUserMode === 'fixed' && authUser) {
                req.auth.user = authUser;
            }
            if (authTokenMode === 'custom') {
                req.auth.tokenMode = 'custom';
                req.auth.token = authToken;
            }
        }


        try {
            setPortActionError(null);
            
            const created = await addPort(req);
            if (created.origin && !created.origin.tcpReachable) {
                setPortActionError(`Mapping saved; origin unreachable from this server: ${created.origin.error}. Tunnel status is not target health.`);
            }
            
            setShowNewPortForm(false);
            setNewPortNumber('');
            setNewPortLabel('');
            setNewPortSubdomain('');
            setNewPortBaseDomain('');
            setRequireAuth(false);
            setAuthUserMode('any');
            setAuthUser('');
            setAuthTokenMode('shared');
            setAuthToken('');
        } catch (err) {
            setPortActionError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleRemovePort = async (port: number | string) => {
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
            console.log('[PortsLayout] onToggleNewForm called, newValue:', newValue);
            setShowNewPortForm(newValue);
            // Generate subdomain when opening form with Cloudflare provider
            // Use ref to get latest provider state (avoid stale closure)
            const currentProvider = providerRef.current;
            console.log('[PortsLayout] currentProvider from ref:', currentProvider);
            if (newValue && (currentProvider === TunnelProviders.CloudflareTunnel || currentProvider === TunnelProviders.CloudflareOwned)) {
                console.log('[PortsLayout] Generating subdomain on form open');
                generateSubdomain(newPortBaseDomain || undefined);
            }
        },
        newTargetHost,
        newTargetScheme,
        onTargetHostChange: (host) => {
            setNewTargetHost(host);
            if (host && !['127.0.0.1', 'localhost', '::1'].includes(host)) {
                setRequireAuth(true); setAuthTokenMode('shared'); setAuthToken('');
            }
        },
        onTargetSchemeChange: setNewTargetScheme,
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
            console.log('[PortsLayout] onPortBaseDomainChange called with:', domain);
            console.log('[PortsLayout] Current state before update:', { newPortBaseDomain, newPortSubdomain, newPortProvider });
            setNewPortBaseDomain(domain);
            console.log('[PortsLayout] After setNewPortBaseDomain, current ref value:', newPortBaseDomain);
            // Generate subdomain when selecting a base domain for Cloudflare providers
            // Use ref to get latest subdomain state (avoid stale closure)
            const currentProvider = providerRef.current;
            const currentSubdomain = subdomainRef.current;
            console.log('[PortsLayout] Ref values - provider:', currentProvider, 'subdomain:', currentSubdomain);
            if (domain && !currentSubdomain && (currentProvider === TunnelProviders.CloudflareTunnel || currentProvider === TunnelProviders.CloudflareOwned)) {
                console.log('[PortsLayout] Generating subdomain for domain:', domain);
                generateSubdomain(domain);
            }
        },
        onGenerateSubdomain: () => generateSubdomain(),
        requireAuth,
        authUserMode,
        authUser,
        authTokenMode,
        authToken,
        onRequireAuthChange: setRequireAuth,
        onAuthUserModeChange: setAuthUserMode,
        onAuthUserChange: setAuthUser,
        onAuthTokenModeChange: setAuthTokenMode,
        onAuthTokenChange: setAuthToken,
        onAddPort: handleAddPort,
        onRemovePort: handleRemovePort,
        navigateToView,
    };

    return <Outlet context={ctx} />;
}
