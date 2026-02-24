export interface CustomAgent {
  id: string;
  name: string;
  description: string;
  mode: 'primary' | 'subagent';
  model?: string;
  tools: Record<string, boolean>;
  permissions?: Record<string, string>;
  hasSystemPrompt: boolean;
}

export interface CustomAgentDetail extends CustomAgent {
  systemPrompt?: string;
}

export interface CreateCustomAgentRequest {
  id?: string;
  name: string;
  description?: string;
  mode?: 'primary' | 'subagent';
  model?: string;
  tools?: Record<string, boolean>;
  permissions?: Record<string, string>;
  template?: string;
  systemPrompt?: string;
}

export interface UpdateCustomAgentRequest {
  name?: string;
  description?: string;
  mode?: 'primary' | 'subagent';
  model?: string;
  tools?: Record<string, boolean>;
  permissions?: Record<string, string>;
  systemPrompt?: string;
}

export interface LaunchCustomAgentRequest {
  projectDir: string;
}

export interface LaunchCustomAgentResponse {
  sessionId: string;
  port: number;
  url: string;
}

export interface AgentTemplate {
  id: string;
  name: string;
  description: string;
  mode: 'primary' | 'subagent';
  tools: Record<string, boolean>;
  permissions: Record<string, string>;
}

export const AGENT_TEMPLATES: AgentTemplate[] = [
  {
    id: 'build',
    name: 'Build',
    description: 'Full development agent with all tools enabled',
    mode: 'primary',
    tools: {
      write: true,
      edit: true,
      bash: true,
      grep: true,
      read: true,
      webfetch: true,
      websearch: true,
    },
    permissions: {},
  },
  {
    id: 'plan',
    name: 'Plan',
    description: 'Planning and analysis - read-only, no changes',
    mode: 'primary',
    tools: {
      read: true,
      grep: true,
      webfetch: true,
      websearch: true,
    },
    permissions: {
      edit: 'deny',
      bash: 'deny',
    },
  },
  {
    id: 'refactor',
    name: 'Refactor',
    description: 'Code refactoring specialist',
    mode: 'subagent',
    tools: {
      read: true,
      edit: true,
      write: true,
      grep: true,
    },
    permissions: {
      bash: 'deny',
    },
  },
  {
    id: 'debug',
    name: 'Debug',
    description: 'Debugging and investigation specialist',
    mode: 'subagent',
    tools: {
      read: true,
      grep: true,
      bash: true,
      webfetch: true,
    },
    permissions: {
      edit: 'ask',
      write: 'ask',
    },
  },
];

export async function fetchCustomAgents(): Promise<CustomAgent[]> {
  const resp = await fetch('/api/custom-agents');
  if (!resp.ok) {
    throw new Error('Failed to fetch custom agents');
  }
  return resp.json();
}

export async function fetchCustomAgent(id: string): Promise<CustomAgentDetail> {
  const resp = await fetch(`/api/custom-agents/${encodeURIComponent(id)}`);
  if (!resp.ok) {
    throw new Error('Failed to fetch custom agent');
  }
  return resp.json();
}

export async function createCustomAgent(
  agent: CreateCustomAgentRequest
): Promise<CustomAgent> {
  const resp = await fetch('/api/custom-agents', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(agent),
  });
  if (!resp.ok) {
    const error = await resp.text();
    throw new Error(error || 'Failed to create custom agent');
  }
  return resp.json();
}

export async function updateCustomAgent(
  id: string,
  agent: UpdateCustomAgentRequest
): Promise<CustomAgent> {
  const resp = await fetch(`/api/custom-agents/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(agent),
  });
  if (!resp.ok) {
    const error = await resp.text();
    throw new Error(error || 'Failed to update custom agent');
  }
  return resp.json();
}

export async function deleteCustomAgent(id: string): Promise<void> {
  const resp = await fetch(`/api/custom-agents/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
  if (!resp.ok) {
    const error = await resp.text();
    throw new Error(error || 'Failed to delete custom agent');
  }
}

export async function launchCustomAgent(
  id: string,
  projectDir: string
): Promise<LaunchCustomAgentResponse> {
  const resp = await fetch(`/api/custom-agents/${encodeURIComponent(id)}/launch`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ projectDir }),
  });
  if (!resp.ok) {
    const error = await resp.text();
    throw new Error(error || 'Failed to launch custom agent');
  }
  return resp.json();
}

export interface CustomAgentSession {
  id: string;
  agent_id: string;
  agent_name: string;
  project_dir: string;
  port: number;
  created_at: string;
  status: string;
}

export async function fetchCustomAgentSessions(): Promise<CustomAgentSession[]> {
  const resp = await fetch('/api/custom-agents/sessions');
  if (!resp.ok) {
    const error = await resp.text();
    throw new Error(error || 'Failed to fetch custom agent sessions');
  }
  return resp.json();
}
