import type { CustomAgent } from '../../../api/customAgents';
import { useV2Context } from '../../V2Context';
import { launchCustomAgent } from '../../../api/customAgents';
import { SettingsIcon, TrashIcon } from '../../icons';

interface AgentCardProps {
  agent: CustomAgent;
  onEdit: (agent: CustomAgent) => void;
  onDelete: (agent: CustomAgent) => void;
}

export function CustomAgentCard({ agent, onEdit, onDelete }: AgentCardProps) {
  const { currentProject } = useV2Context();
  const projectDir = currentProject?.dir;

  const handleLaunch = async () => {
    if (!projectDir) return;
    try {
      const result = await launchCustomAgent(agent.id, projectDir);
      window.open(result.url, '_blank');
    } catch (err) {
      console.error('Failed to launch agent:', err);
    }
  };

  const toolCount = agent.tools ? Object.values(agent.tools).filter(Boolean).length : 0;

  return (
    <div className="mcc-agent-card">
      <div className="mcc-agent-card-header">
        <div className="mcc-agent-card-info">
          <span className="mcc-agent-card-name">{agent.name}</span>
          <span className="mcc-agent-card-status installed">Custom</span>
          <span className="mcc-agent-card-status">{agent.mode}</span>
        </div>
        <div className="mcc-agent-card-actions-inline">
          <button
            className="mcc-agent-card-settings-icon"
            onClick={() => onEdit(agent)}
            title="Edit"
          >
            <SettingsIcon />
          </button>
          <button
            className="mcc-agent-card-settings-icon"
            onClick={() => onDelete(agent)}
            title="Delete"
          >
            <TrashIcon />
          </button>
        </div>
      </div>
      <div className="mcc-agent-card-desc">{agent.description}</div>
      <div className="mcc-agent-card-meta">
        {agent.hasSystemPrompt && <span className="mcc-agent-meta-prompt">Has system prompt</span>}
        <span className="mcc-agent-meta-tools">{toolCount} tools</span>
      </div>
      <div className="mcc-agent-card-actions">
        <button
          className="mcc-forward-btn mcc-agent-launch-btn"
          onClick={handleLaunch}
          disabled={!projectDir}
        >
          Start Chat
        </button>
      </div>
    </div>
  );
}
