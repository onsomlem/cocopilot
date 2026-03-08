"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProjectTreeProvider = exports.TaskTreeProvider = void 0;
const vscode = __importStar(require("vscode"));
function getApiBase() {
    return (vscode.workspace.getConfiguration("cocopilot").get("apiBase") ||
        "http://localhost:8080");
}
function getProjectId() {
    return (vscode.workspace.getConfiguration("cocopilot").get("projectId") ||
        "proj_default");
}
function getApiHeaders() {
    const apiKey = vscode.workspace.getConfiguration("cocopilot").get("apiKey") || "";
    const headers = {
        "Content-Type": "application/json",
    };
    if (apiKey.trim()) {
        headers["X-API-Key"] = apiKey.trim();
    }
    return headers;
}
// ---------- Task Tree ----------
class TaskItem extends vscode.TreeItem {
    constructor(task) {
        super(`#${task.id} ${task.title}`, vscode.TreeItemCollapsibleState.None);
        this.task = task;
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
    getStatusIcon(status) {
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
class TaskTreeProvider {
    constructor() {
        this._onDidChangeTreeData = new vscode.EventEmitter();
        this.onDidChangeTreeData = this._onDidChangeTreeData.event;
        this.tasks = [];
    }
    refresh() {
        this._onDidChangeTreeData.fire(undefined);
    }
    getTreeItem(element) {
        return element;
    }
    async getChildren() {
        try {
            const base = getApiBase();
            const projectId = getProjectId();
            const url = `${base}/api/v2/projects/${encodeURIComponent(projectId)}/tasks?limit=100`;
            const resp = await fetch(url, { headers: getApiHeaders() });
            if (!resp.ok) {
                return [];
            }
            const data = (await resp.json());
            this.tasks = Array.isArray(data.tasks) ? data.tasks : [];
            return this.tasks.map((t) => new TaskItem(t));
        }
        catch {
            return [];
        }
    }
}
exports.TaskTreeProvider = TaskTreeProvider;
// ---------- Project Tree ----------
class ProjectItem extends vscode.TreeItem {
    constructor(project) {
        super(project.name, vscode.TreeItemCollapsibleState.None);
        this.project = project;
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
class ProjectTreeProvider {
    constructor() {
        this._onDidChangeTreeData = new vscode.EventEmitter();
        this.onDidChangeTreeData = this._onDidChangeTreeData.event;
    }
    refresh() {
        this._onDidChangeTreeData.fire(undefined);
    }
    getTreeItem(element) {
        return element;
    }
    async getChildren() {
        try {
            const base = getApiBase();
            const url = `${base}/api/v2/projects`;
            const resp = await fetch(url, { headers: getApiHeaders() });
            if (!resp.ok) {
                return [];
            }
            const data = (await resp.json());
            const projects = Array.isArray(data.projects) ? data.projects : [];
            return projects.map((p) => new ProjectItem(p));
        }
        catch {
            return [];
        }
    }
}
exports.ProjectTreeProvider = ProjectTreeProvider;
//# sourceMappingURL=sidebar.js.map