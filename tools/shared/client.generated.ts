/**
 * AUTO-GENERATED typed API client -- DO NOT EDIT.
 * Source: tools/codegen/codegen.go
 */

import type {
  Project, TaskV2, Run, Event, Agent, Memory, Policy,
  AssignmentEnvelope, ContextPack, RepoFile, DashboardData,
  TaskTemplate,
} from './types.generated';

export interface ClientOptions {
  baseURL: string;
  apiKey?: string;
}

export class CocopilotClient {
  private baseURL: string;
  private headers: Record<string, string>;

  constructor(opts: ClientOptions) {
    this.baseURL = opts.baseURL.replace(/\/+$/, '');
    this.headers = { 'Content-Type': 'application/json' };
    if (opts.apiKey) {
      this.headers['X-API-Key'] = opts.apiKey;
    }
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const url = this.baseURL + path;
    const init: RequestInit = { method, headers: this.headers };
    if (body !== undefined) {
      init.body = JSON.stringify(body);
    }
    const resp = await fetch(url, init);
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`HTTP ${resp.status}: ${text}`);
    }
    if (resp.status === 204) return undefined as T;
    return resp.json() as Promise<T>;
  }

  // Health
  health() { return this.request<{ ok: boolean }>('GET', '/api/v2/health'); }

  // Projects
  listProjects() { return this.request<{ projects: Project[] }>('GET', '/api/v2/projects'); }
  getProject(id: string) { return this.request<Project>('GET', `/api/v2/projects/${id}`); }
  createProject(name: string, workdir: string) {
    return this.request<Project>('POST', '/api/v2/projects', { name, workdir });
  }

  // Tasks
  listTasks(projectId: string, params?: Record<string, string>) {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<{ tasks: TaskV2[]; total: number }>('GET', `/api/v2/projects/${projectId}/tasks${qs}`);
  }
  getTask(id: number) { return this.request<TaskV2>('GET', `/api/v2/tasks/${id}`); }
  createTask(projectId: string, body: Partial<TaskV2>) {
    return this.request<TaskV2>('POST', `/api/v2/projects/${projectId}/tasks`, body);
  }
  updateTask(id: number, body: Partial<TaskV2>) {
    return this.request<TaskV2>('PATCH', `/api/v2/tasks/${id}`, body);
  }
  deleteTask(id: number) { return this.request<void>('DELETE', `/api/v2/tasks/${id}`); }

  // Claims
  claimTask(id: number, agentId: string) {
    return this.request<AssignmentEnvelope>('POST', `/api/v2/tasks/${id}/claim`, { agent_id: agentId });
  }
  claimNext(projectId: string, agentId: string) {
    return this.request<AssignmentEnvelope>('POST', `/api/v2/projects/${projectId}/tasks/claim-next`, { agent_id: agentId });
  }
  completeTask(id: number, body: Record<string, unknown>) {
    return this.request<void>('POST', `/api/v2/tasks/${id}/complete`, body);
  }
  failTask(id: number, body: Record<string, unknown>) {
    return this.request<void>('POST', `/api/v2/tasks/${id}/fail`, body);
  }

  // Runs
  getRun(id: string) { return this.request<Run>('GET', `/api/v2/runs/${id}`); }

  // Events
  listEvents(params?: Record<string, string>) {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<{ events: Event[] }>('GET', `/api/v2/events${qs}`);
  }

  // Agents
  listAgents() { return this.request<{ agents: Agent[] }>('GET', '/api/v2/agents'); }

  // Memory
  listMemories(projectId: string) {
    return this.request<{ memories: Memory[] }>('GET', `/api/v2/projects/${projectId}/memory`);
  }
  putMemory(projectId: string, body: Partial<Memory>) {
    return this.request<Memory>('PUT', `/api/v2/projects/${projectId}/memory`, body);
  }

  // Policies
  listPolicies(projectId: string) {
    return this.request<{ policies: Policy[] }>('GET', `/api/v2/projects/${projectId}/policies`);
  }

  // Dashboard
  getDashboard(projectId: string) {
    return this.request<DashboardData>('GET', `/api/v2/projects/${projectId}/dashboard`);
  }

  // Repo Files
  listRepoFiles(projectId: string) {
    return this.request<{ files: RepoFile[]; total: number }>('GET', `/api/v2/projects/${projectId}/files`);
  }
  scanRepoFiles(projectId: string) {
    return this.request<{ files: RepoFile[] }>('POST', `/api/v2/projects/${projectId}/files/scan`);
  }

  // Context Packs
  listContextPacks(projectId: string) {
    return this.request<{ context_packs: ContextPack[] }>('GET', `/api/v2/projects/${projectId}/context-packs`);
  }

  // Templates
  listTemplates(projectId: string) {
    return this.request<{ templates: TaskTemplate[] }>('GET', `/api/v2/projects/${projectId}/templates`);
  }

  // Notifications
  listNotifications(projectId: string, params?: Record<string, string>) {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<{ events: Event[] }>('GET', `/api/v2/projects/${projectId}/notifications${qs}`);
  }
}
