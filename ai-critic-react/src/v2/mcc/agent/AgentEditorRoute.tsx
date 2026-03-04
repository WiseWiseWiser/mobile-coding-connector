import { useNavigate, useOutletContext, useParams } from 'react-router-dom';
import type { AgentOutletContext } from './AgentLayout';
import { AgentEditor } from './AgentEditor';
import { useState, useEffect } from 'react';
import { fetchCustomAgent } from '../../../api/customAgents';
import type { CustomAgent } from '../../../api/customAgents';

export function AgentEditorRoute() {
    const navigate = useNavigate();
    const ctx = useOutletContext<AgentOutletContext>();
    const { agentId } = useParams<{ agentId: string }>();
    const [agent, setAgent] = useState<CustomAgent | null>(null);
    const [loading, setLoading] = useState(!!agentId);

    const isEdit = !!agentId;

    useEffect(() => {
        if (!agentId) return;
        setLoading(true);
        fetchCustomAgent(agentId)
            .then(setAgent)
            .catch(() => setAgent(null))
            .finally(() => setLoading(false));
    }, [agentId]);

    const handleSave = () => {
        navigate(isEdit ? '../..' : '..', { relative: 'path' });
        ctx.onRefreshAgents();
    };

    const handleCancel = () => {
        navigate(isEdit ? '../..' : '..', { relative: 'path' });
    };

    if (loading) {
        return <div className="mcc-agent-loading">Loading agent...</div>;
    }

    return (
        <div className="mcc-agent-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={handleCancel}>
                    &larr;
                </button>
                <h2>{isEdit ? 'Edit Agent' : 'New Agent'}</h2>
            </div>
            <AgentEditor
                agent={isEdit ? agent : null}
                onSave={handleSave}
                onCancel={handleCancel}
            />
        </div>
    );
}
