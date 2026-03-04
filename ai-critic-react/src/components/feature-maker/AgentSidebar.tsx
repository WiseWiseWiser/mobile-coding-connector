import { PathInput } from '../../pure-view/PathInput';
import { mockMainAgent, mockSubAgents } from '../../mockups/featureMakerMockData';
import './AgentSidebar.css';

interface AgentSidebarProps {
    showPathInput: boolean;
    projectPath: string;
    onProjectPathChange: (path: string) => void;
}

export function AgentSidebar({ showPathInput, projectPath, onProjectPathChange }: AgentSidebarProps) {
    return (
        <div className="fm-sidebar">
            {showPathInput && (
                <div className="fm-project-path">
                    <PathInput
                        value={projectPath}
                        onChange={onProjectPathChange}
                        label="Project Directory"
                    />
                </div>
            )}

            <div className="fm-main-agent">
                <h3>Main Agent</h3>
                <div className="fm-agent-name">{mockMainAgent.name}</div>

                <div className="fm-agent-prompt">
                    <h4>System Prompt</h4>
                    <div className="fm-prompt-content">
                        {mockMainAgent.systemPrompt.split('\n\n').map((section, i) => {
                            const lines = section.split('\n');
                            const title = lines[0].replace(/^#\s*/, '');
                            const content = lines.slice(1).join('\n');
                            return (
                                <div key={i} className="fm-prompt-section">
                                    <span className="fm-prompt-section-title">{title}</span>
                                    <span className="fm-prompt-section-content">{content}</span>
                                </div>
                            );
                        })}
                    </div>
                </div>

                <div className="fm-agent-tools">
                    <h4>Available Tools</h4>
                    <div className="fm-tools-list">
                        {mockMainAgent.tools.map((tool, i) => (
                            <div key={i} className="fm-tool-item">
                                <span className="fm-tool-name">{tool.name}</span>
                                <span className="fm-tool-desc">{tool.description}</span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            <div className="fm-subagents">
                <h3>Subagents</h3>
                <div className="fm-agent-list">
                    {mockSubAgents.map(agent => (
                        <div key={agent.name} className={`fm-agent ${agent.status}`}>
                            <span className="fm-agent-status-dot"></span>
                            <span className="fm-agent-name">{agent.name}</span>
                            <span className="fm-agent-role">{agent.role}</span>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}
