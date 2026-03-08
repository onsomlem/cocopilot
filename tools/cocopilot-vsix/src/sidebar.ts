import * as vscode from "vscode";

interface CocopilotTask {
  id: number;
  title: string;
  status: string;
  type?: string;
  priority?: number;
  instructions?: string;
  tags?: string[];
  created_at?: string;
  updated_at?: string;
}

interface CocopilotProject {
  id: string;
  name: string;
  workdir?: string;
  description?: string;
}

function getApiBase(): string {
  return (
    vscode.workspace.getConfiguration("cocopilot").get<string>("apiBase") ||
    "http://localhost:8080"
  );
}

function getProjectId(): string {
  return (
    vscode.workspace.getConfiguration("cocopilot").get<string>("projectId") ||
    "proj_default"
  );
}

function getApiHeaders(): Record<string, string> {
  const apiKey =
    vscode.workspace.getConfiguration("cocopilot").get<string>("apiKey") || "";
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (apiKey.trim()) {
    headers["X-API-Key"] = apiKey.trim();
  }
  return headers;
}

// ---------- Task Tree ----------

class TaskItem extends vscode.TreeItem {
  constructor(public readonly task: CocopilotTask) {
    super(
      `#${task.id} ${task.title}`,
      vscode.TreeItemCollapsibleState.None
    );
    this.tooltip = `${task.title}\nStatus: ${task.status}\nType: ${task.type || "—"}\nPriority: ${task.priority ?? "—"}`;
    this.description = task.status;
    this.contextValue = "task";

    const statusIcon = this.getStatusIcon(task.status);
    this.iconPath = new vscode.ThemeIcon(statusIcon.icon, statusIcon.color);

    this.command = {
      command: "cocopilot.sidebarTaskDetail",
      title: "Show Task",
      arguments: [task],
    };
  }

  private getStatusIcon(status: string): {
    icon: string;
    color?: vscode.ThemeColor;
  } {
    switch (status?.toUpperCase()) {
      case "DONE":
      case "SUCCEEDED":
        return {
          icon: "pass-filled",
          color: new vscode.ThemeColor("testing.iconPassed"),
        };
      case "IN_PROGRESS":
      case "RUNNING":
        return {
          icon: "sync~spin",
          color: new vscode.ThemeColor("charts.blue"),
        };
      case "FAILED":
        return {
          icon: "error",
          color: new vscode.ThemeColor("testing.iconFailed"),
        };
      case "BLOCKED":
        return {
          icon: "lock",
          color: new vscode.ThemeColor("errorForeground"),
        };
      case "CANCELED":
        return {
          icon: "circle-slash",
          color: new vscode.ThemeColor("disabledForeground"),
        };
      case "REVIEW":
        return {
          icon: "eye",
          color: new vscode.ThemeColor("charts.yellow"),
        };
      default:
        return {
          icon: "circle-outline",
          color: new vscode.ThemeColor("disabledForeground"),
        };
    }
  }
}

export class TaskTreeProvider
  implements vscode.TreeDataProvider<TaskItem>
{
  private _onDidChangeTreeData = new vscode.EventEmitter<
    TaskItem | undefined | null
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private tasks: CocopilotTask[] = [];

  refresh(): void {
    this._onDidChangeTreeData.fire(undefined);
  }

  getTreeItem(element: TaskItem): vscode.TreeItem {
    return element;
  }

  async getChildren(): Promise<TaskItem[]> {
    try {
      const base = getApiBase();
      const projectId = getProjectId();
      const url = `${base}/api/v2/projects/${encodeURIComponent(projectId)}/tasks?limit=100`;
      const resp = await fetch(url, { headers: getApiHeaders() });
      if (!resp.ok) {
        return [];
      }
      const data = (await resp.json()) as { tasks?: CocopilotTask[] };
      this.tasks = Array.isArray(data.tasks) ? data.tasks : [];
      return this.tasks.map((t) => new TaskItem(t));
    } catch {
      return [];
    }
  }
}

// ---------- Project Tree ----------

class ProjectItem extends vscode.TreeItem {
  constructor(public readonly project: CocopilotProject) {
    super(project.name, vscode.TreeItemCollapsibleState.None);
    this.tooltip = `${project.name}\nID: ${project.id}${project.workdir ? "\nWorkdir: " + project.workdir : ""}`;
    this.description = project.id;
    this.contextValue = "project";
    this.iconPath = new vscode.ThemeIcon("folder");

    this.command = {
      command: "cocopilot.sidebarSelectProject",
      title: "Select Project",
      arguments: [project],
    };
  }
}

export class ProjectTreeProvider
  implements vscode.TreeDataProvider<ProjectItem>
{
  private _onDidChangeTreeData = new vscode.EventEmitter<
    ProjectItem | undefined | null
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  refresh(): void {
    this._onDidChangeTreeData.fire(undefined);
  }

  getTreeItem(element: ProjectItem): vscode.TreeItem {
    return element;
  }

  async getChildren(): Promise<ProjectItem[]> {
    try {
      const base = getApiBase();
      const url = `${base}/api/v2/projects`;
      const resp = await fetch(url, { headers: getApiHeaders() });
      if (!resp.ok) {
        return [];
      }
      const data = (await resp.json()) as { projects?: CocopilotProject[] };
      const projects = Array.isArray(data.projects) ? data.projects : [];
      return projects.map((p) => new ProjectItem(p));
    } catch {
      return [];
    }
  }
}
