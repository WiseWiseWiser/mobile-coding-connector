import { useState, useEffect } from 'react';
import type { CustomAgent, CreateCustomAgentRequest } from '../../../api/customAgents';
import {
  AGENT_TEMPLATES,
  createCustomAgent,
  updateCustomAgent,
  fetchCustomAgent,
} from '../../../api/customAgents';

interface AgentEditorProps {
  agent?: CustomAgent | null;
  onSave: () => void;
  onCancel: () => void;
}

const ALL_TOOLS = [
  { key: 'write', label: 'Write Files' },
  { key: 'edit', label: 'Edit Files' },
  { key: 'bash', label: 'Bash Commands' },
  { key: 'grep', label: 'Search Code' },
  { key: 'read', label: 'Read Files' },
  { key: 'webfetch', label: 'Fetch Web Content' },
  { key: 'websearch', label: 'Web Search' },
];

const ALL_PERMISSIONS = [
  { key: 'edit', label: 'File Edits' },
  { key: 'bash', label: 'Bash Commands' },
];

export function AgentEditor({ agent, onSave, onCancel }: AgentEditorProps) {
  const [name, setName] = useState(agent?.name || '');
  const [description, setDescription] = useState(agent?.description || '');
  const [mode, setMode] = useState<'primary' | 'subagent'>(agent?.mode || 'primary');
  const [model, setModel] = useState(agent?.model || '');
  const [tools, setTools] = useState<Record<string, boolean>>(agent?.tools || {});
  const [permissions, setPermissions] = useState<Record<string, string>>(agent?.permissions || {});
  const [systemPrompt, setSystemPrompt] = useState('');
  const [template, setTemplate] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const isEdit = !!agent;

  useEffect(() => {
    if (agent?.id) {
      fetchCustomAgent(agent.id)
        .then((detail) => {
          setSystemPrompt(detail.systemPrompt || '');
        })
        .catch(console.error);
    }
  }, [agent?.id]);

  const handleTemplateChange = (templateId: string) => {
    setTemplate(templateId);
    const tmpl = AGENT_TEMPLATES.find((t) => t.id === templateId);
    if (tmpl) {
      if (!name) setName(tmpl.name);
      if (!description) setDescription(tmpl.description);
      if (!mode) setMode(tmpl.mode);
      setTools({ ...tmpl.tools });
      setPermissions({ ...tmpl.permissions });
      setSystemPrompt(tmpl.id === 'build' ? getBuildPrompt() : tmpl.id === 'plan' ? getPlanPrompt() : tmpl.id === 'refactor' ? getRefactorPrompt() : tmpl.id === 'debug' ? getDebugPrompt() : '');
    }
  };

  const handleToolChange = (key: string, checked: boolean) => {
    setTools((prev) => ({ ...prev, [key]: checked }));
  };

  const handlePermissionChange = (key: string, value: string) => {
    setPermissions((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);

    try {
      const req: CreateCustomAgentRequest = {
        name,
        description,
        mode,
        model: model || undefined,
        tools,
        permissions,
        systemPrompt: systemPrompt || undefined,
      };

      if (isEdit && agent) {
        await updateCustomAgent(agent.id, req);
      } else {
        req.id = name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
        await createCustomAgent(req);
      }

      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save agent');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mcc-agent-editor">
      <div className="mcc-agent-editor-header">
        <h3>{isEdit ? 'Edit Agent' : 'New Agent'}</h3>
      </div>

      <form onSubmit={handleSubmit}>
        {!isEdit && (
          <div className="mcc-form-group">
            <label>Template (optional)</label>
            <select
              value={template}
              onChange={(e) => handleTemplateChange(e.target.value)}
              className="mcc-select"
            >
              <option value="">Select a template...</option>
              {AGENT_TEMPLATES.map((tmpl) => (
                <option key={tmpl.id} value={tmpl.id}>
                  {tmpl.name} - {tmpl.description}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="mcc-form-group">
          <label>Name *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="mcc-input"
            placeholder="My Agent"
            required
          />
        </div>

        <div className="mcc-form-group">
          <label>Description</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="mcc-input"
            placeholder="What this agent does..."
          />
        </div>

        <div className="mcc-form-group">
          <label>Mode</label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as 'primary' | 'subagent')}
            className="mcc-select"
          >
            <option value="primary">Primary Agent</option>
            <option value="subagent">Subagent</option>
          </select>
        </div>

        <div className="mcc-form-group">
          <label>Model (optional)</label>
          <input
            type="text"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className="mcc-input"
            placeholder="e.g., anthropic/claude-sonnet-4-5"
          />
          <small>Leave empty to use default model</small>
        </div>

        <div className="mcc-form-group">
          <label>Tools</label>
          <div className="mcc-checkbox-grid">
            {ALL_TOOLS.map((tool) => (
              <label key={tool.key} className="mcc-checkbox-label">
                <input
                  type="checkbox"
                  checked={tools[tool.key] || false}
                  onChange={(e) => handleToolChange(tool.key, e.target.checked)}
                />
                {tool.label}
              </label>
            ))}
          </div>
        </div>

        <div className="mcc-form-group">
          <label>Permissions</label>
          <div className="mcc-permission-grid">
            {ALL_PERMISSIONS.map((perm) => (
              <div key={perm.key} className="mcc-permission-row">
                <span>{perm.label}</span>
                <select
                  value={permissions[perm.key] || 'allow'}
                  onChange={(e) => handlePermissionChange(perm.key, e.target.value)}
                  className="mcc-select-small"
                >
                  <option value="allow">Allow</option>
                  <option value="ask">Ask</option>
                  <option value="deny">Deny</option>
                </select>
              </div>
            ))}
          </div>
        </div>

        <div className="mcc-form-group">
          <label>System Prompt</label>
          <textarea
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            className="mcc-textarea"
            placeholder="Instructions for the agent..."
            rows={10}
          />
        </div>

        {error && <div className="mcc-error">{error}</div>}

        <div className="mcc-form-actions">
          <button type="button" onClick={onCancel} className="mcc-btn-secondary">
            Cancel
          </button>
          <button type="submit" disabled={saving} className="mcc-btn-primary">
            {saving ? 'Saving...' : isEdit ? 'Update' : 'Create'}
          </button>
        </div>
      </form>
    </div>
  );
}

function getBuildPrompt(): string {
  return `# Build Agent

You are a coding assistant focused on implementing features and making changes to the codebase.

## Your Role
- Implement new features based on user requirements
- Make code changes as requested
- Write clean, maintainable code
- Follow the project's coding conventions

## Guidelines
- Always ask for clarification if requirements are unclear
- Before making significant changes, explain your approach
- Write tests when appropriate
- Ensure code is properly formatted
`;
}

function getPlanPrompt(): string {
  return `# Plan Agent

You are a planning and analysis agent. You analyze code and create plans without making any changes.

## Your Role
- Analyze existing code and understand its structure
- Create implementation plans
- Suggest improvements and refactoring
- Review and critique proposed changes

## Guidelines
- Do NOT make any code changes
- Provide detailed analysis of the codebase
- Create step-by-step plans for implementation
- Consider edge cases and potential issues
- Suggest best practices and design patterns
`;
}

function getRefactorPrompt(): string {
  return `# Refactor Agent

You are a code refactoring specialist focused on improving code quality without changing external behavior.

## Your Role
- Refactor code to improve readability and maintainability
- Extract reusable components
- Simplify complex logic
- Apply design patterns where appropriate
- Eliminate code duplication

## Guidelines
- Maintain the same external behavior
- Make small, incremental changes
- Ensure refactored code passes existing tests
- Focus on one refactoring at a time
- Explain the benefits of each refactoring
`;
}

function getDebugPrompt(): string {
  return `# Debug Agent

You are a debugging and investigation specialist focused on finding and fixing issues in the codebase.

## Your Role
- Investigate bugs and errors
- Find root causes of issues
- Analyze logs and error messages
- Propose fixes for identified problems

## Guidelines
- Start by understanding the error or unexpected behavior
- Trace the issue through the codebase
- Look for common bug patterns
- Verify your findings with tests or manual verification
- Provide clear explanation of the root cause
- Suggest fixes, but implement them only when explicitly requested
`;
}
