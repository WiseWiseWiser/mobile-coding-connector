import React, { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { PortTargetFields } from '../../src/v2/mcc/PortTargetFields';
import { PortForwardCard } from '../../src/v2/mcc/PortForwardCard';
import { removePort } from '../../src/api/ports';
import { TunnelProviders } from '../../src/hooks/usePortForwards';

function Fixture() {
    const [host, setHost] = useState('192.168.1.1');
    const [scheme, setScheme] = useState<'http' | 'https'>('http');
    return <>
        <div data-testid="fields"><PortTargetFields host={host} scheme={scheme} port="3080" onHostChange={setHost} onSchemeChange={setScheme} /></div>
        {['map_pi', 'map_mac'].map(id => <div data-testid={id} key={id}><PortForwardCard port={{ id, localPort: 3080, target: { scheme: 'http', host: id === 'map_pi' ? '127.0.0.1' : '192.168.1.1', port: 3080 }, label: `${id}.example`, publicUrl: '', status: 'active', provider: TunnelProviders.CloudflareOwned }} onRemove={() => { void removePort(id); }} onNavigateToView={() => {}} /></div>)}
    </>;
}

createRoot(document.getElementById('root')!).render(<Fixture />);
