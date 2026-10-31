import React from 'react';
import { createRoot } from 'react-dom/client';
import { PortForwardCard } from '../../src/v2/mcc/PortForwardCard';
import { TunnelProviders } from '../../src/hooks/usePortForwards';

function Fixture() {
    const [removed, setRemoved] = React.useState(0);
    return <>
        <div data-testid="bootstrap"><PortForwardCard port={{localPort:23712,label:'bootstrap.example',publicUrl:'https://bootstrap.example',status:'error',error:'DNS unavailable',provider:TunnelProviders.CloudflareOwned,bootstrap:true,type:'domain'}} onRemove={() => setRemoved(n => n + 100)} onNavigateToView={() => {}} /></div>
        <div data-testid="normal"><PortForwardCard port={{localPort:8080,label:'normal.example',publicUrl:'https://normal.example',status:'active',provider:TunnelProviders.CloudflareOwned}} onRemove={() => setRemoved(n => n + 1)} onNavigateToView={() => {}} /></div>
        <output data-testid="removed">{removed}</output>
    </>;
}

createRoot(document.getElementById('root')!).render(<Fixture />);
