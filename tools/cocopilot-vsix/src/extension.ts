import * as vscode from "vscode";

export function activate(context: vscode.ExtensionContext) {
  console.log("cocopilot-vsix activated");

  let mcpTerminal: vscode.Terminal | undefined;
  const outputChannel = vscode.window.createOutputChannel("Cocopilot");

  const logMcp = (message: string) => {
    outputChannel.appendLine(`[MCP] ${new Date().toISOString()} ${message}`);
  };

  const logMcpError = (message: string, err: unknown) => {
    const detail = err instanceof Error ? err.message : String(err);
    outputChannel.appendLine(`[MCP] ${new Date().toISOString()} ${message}: ${detail}`);
  };

  // Returns HTTP headers including X-API-Key if configured.
  const getApiHeaders = (extra?: Record<string, string>): Record<string, string> => {
    const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
    const apiKey = cocopilotConfig.get<string>("apiKey") || "";
    const headers: Record<string, string> = { ...extra };
    if (apiKey.trim()) {
      headers["X-API-Key"] = apiKey.trim();
    }
    return headers;
  };

  const statusBar = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Left,
    100
  );
  statusBar.command = "cocopilot.openSettings";
  statusBar.tooltip = "Edit Cocopilot settings";

  const mcpStatusBar = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Left,
    99
  );

  const updateStatusBar = () => {
    const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
    const projectId = cocopilotConfig.get<string>("projectId") || "";
    const label = projectId.trim() ? projectId.trim() : "unset";
    statusBar.text = `Cocopilot: ${label}`;
    statusBar.show();
  };

  const updateMcpStatusBar = (isRunning: boolean) => {
    if (isRunning) {
      mcpStatusBar.text = "MCP: Running";
      mcpStatusBar.command = "cocopilot.stopMcpServer";
      mcpStatusBar.tooltip = "Stop Cocopilot MCP server";
    } else {
      mcpStatusBar.text = "MCP: Stopped";
      mcpStatusBar.command = "cocopilot.startMcpServer";
      mcpStatusBar.tooltip = "Start Cocopilot MCP server";
    }
    mcpStatusBar.show();
  };

  const disposable = vscode.commands.registerCommand(
    "cocopilot.hello",
    () => {
      vscode.window.showInformationMessage("Hello from Cocopilot");
    }
  );

  const configureMcp = vscode.commands.registerCommand(
    "cocopilot.configureMcp",
    async () => {
      logMcp("configure: start");
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        logMcp("configure: missing workspace folder");
        return;
      }

      try {
        const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
        const apiBase = cocopilotConfig.get<string>("apiBase") || "<COCO_API_BASE>";
        const projectId = cocopilotConfig.get<string>("projectId") || "<COCO_PROJECT_ID>";
        const apiKey = cocopilotConfig.get<string>("apiKey") || "";

        const vscodeDir = vscode.Uri.joinPath(workspaceFolder.uri, ".vscode");
        await vscode.workspace.fs.createDirectory(vscodeDir);

        const configUri = vscode.Uri.joinPath(vscodeDir, "mcp.json");
        const config = {
          servers: {
            cocopilot: {
              command: "node",
              args: ["tools/cocopilot-mcp/dist/index.js"],
              env: {
                COCO_API_BASE: apiBase,
                COCO_PROJECT_ID: projectId,
                ...(apiKey.trim() ? { COCO_API_KEY: apiKey.trim() } : {})
              }
            }
          }
        };

        const content = JSON.stringify(config, null, 2);
        await vscode.workspace.fs.writeFile(configUri, Buffer.from(content, "utf8"));
        vscode.window.showInformationMessage("Wrote .vscode/mcp.json for Cocopilot MCP.");
        logMcp("configure: wrote .vscode/mcp.json");
      } catch (err) {
        vscode.window.showErrorMessage("Failed to write .vscode/mcp.json for Cocopilot MCP.");
        logMcpError("configure: failed", err);
      }
    }
  );

  const buildMcpConfig = () => {
    const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
    const apiBase = cocopilotConfig.get<string>("apiBase") || "<COCO_API_BASE>";
    const projectId = cocopilotConfig.get<string>("projectId") || "<COCO_PROJECT_ID>";
    const apiKey = cocopilotConfig.get<string>("apiKey") || "";
    return {
      servers: {
        cocopilot: {
          command: "node",
          args: ["tools/cocopilot-mcp/dist/index.js"],
          env: {
            COCO_API_BASE: apiBase,
            COCO_PROJECT_ID: projectId,
            ...(apiKey.trim() ? { COCO_API_KEY: apiKey.trim() } : {})
          }
        }
      }
    };
  };

  const ensureMcpConfig = async (workspaceFolder: vscode.WorkspaceFolder) => {
    const vscodeDir = vscode.Uri.joinPath(workspaceFolder.uri, ".vscode");
    await vscode.workspace.fs.createDirectory(vscodeDir);
    const configUri = vscode.Uri.joinPath(vscodeDir, "mcp.json");
    try {
      await vscode.workspace.fs.stat(configUri);
    } catch {
      const content = JSON.stringify(buildMcpConfig(), null, 2);
      await vscode.workspace.fs.writeFile(configUri, Buffer.from(content, "utf8"));
    }
    return configUri;
  };

  const openMcpConfig = vscode.commands.registerCommand(
    "cocopilot.openMcpConfig",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const configUri = await ensureMcpConfig(workspaceFolder);
      const doc = await vscode.workspace.openTextDocument(configUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const startMcpServerNow = async (showAlreadyRunningMessage: boolean) => {
    logMcp("start: requested");
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    if (!workspaceFolder) {
      vscode.window.showErrorMessage("Open a workspace folder first.");
      logMcp("start: missing workspace folder");
      return;
    }

    const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
    const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
    const projectId = cocopilotConfig.get<string>("projectId") || "";
    if (mcpTerminal) {
      mcpTerminal.show();
      if (showAlreadyRunningMessage) {
        vscode.window.showInformationMessage("Cocopilot MCP server is already running.");
      }
      updateMcpStatusBar(true);
      logMcp("start: already running");
      return;
    }

    const terminal = vscode.window.createTerminal({
      name: "Cocopilot MCP",
      cwd: vscode.Uri.joinPath(workspaceFolder.uri, "tools", "cocopilot-mcp").fsPath,
      env: {
        COCO_API_BASE: apiBase,
        COCO_PROJECT_ID: projectId
      }
    });
    mcpTerminal = terminal;
    terminal.show();
    terminal.sendText("npm run start", true);
    updateMcpStatusBar(true);
    logMcp("start: launched terminal");
  };

  const startMcpServer = vscode.commands.registerCommand(
    "cocopilot.startMcpServer",
    async () => {
      await startMcpServerNow(true);
    }
  );

  const stopMcpServer = vscode.commands.registerCommand(
    "cocopilot.stopMcpServer",
    async () => {
      logMcp("stop: requested");
      if (!mcpTerminal) {
        vscode.window.showInformationMessage("Cocopilot MCP server is not running.");
        logMcp("stop: not running");
        return;
      }

      mcpTerminal.dispose();
      mcpTerminal = undefined;
      vscode.window.showInformationMessage("Stopped Cocopilot MCP server.");
      updateMcpStatusBar(false);
      logMcp("stop: terminal disposed");
    }
  );

  const buildUiUrl = (pathSuffix?: string) => {
    const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
    const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
    const projectId = cocopilotConfig.get<string>("projectId") || "";
    const trimmedProjectId = projectId.trim();
    const normalizedSuffix = pathSuffix ? pathSuffix.trim() : "";
    let targetUrl = "";

    try {
      const baseUrl = new URL(apiBase);
      const target = normalizedSuffix ? new URL(normalizedSuffix, baseUrl) : baseUrl;
      if (trimmedProjectId) {
        target.searchParams.set("project_id", trimmedProjectId);
      }
      targetUrl = target.toString();
    } catch {
      const trimmedBase = apiBase.replace(/\/+$/, "");
      const suffix = normalizedSuffix ? normalizedSuffix.replace(/^\/+/, "") : "";
      const baseWithPath = suffix ? `${trimmedBase}/${suffix}` : trimmedBase;
      if (trimmedProjectId) {
        const separator = baseWithPath.includes("?") ? "&" : "?";
        targetUrl = `${baseWithPath}${separator}project_id=${encodeURIComponent(trimmedProjectId)}`;
      } else {
        targetUrl = baseWithPath;
      }
    }

    return targetUrl;
  };

  const openDashboard = vscode.commands.registerCommand(
    "cocopilot.openDashboard",
    async () => {
      const targetUrl = buildUiUrl();
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openTasksBoard = vscode.commands.registerCommand(
    "cocopilot.openTasksBoard",
    async () => {
      const targetUrl = buildUiUrl("/");
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openHealth = vscode.commands.registerCommand(
    "cocopilot.openHealth",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL("/api/v2/health", baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/health`;
      }

      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openConfig = vscode.commands.registerCommand(
    "cocopilot.openConfig",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL("/api/v2/config", baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/config`;
      }

      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openVersion = vscode.commands.registerCommand(
    "cocopilot.openVersion",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL("/api/v2/version", baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/version`;
      }

      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openEvents = vscode.commands.registerCommand(
    "cocopilot.openEvents",
    async () => {
      const targetUrl = buildUiUrl("/api/v2/events");
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openEventsStream = vscode.commands.registerCommand(
    "cocopilot.openEventsStream",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const defaultProjectId = cocopilotConfig.get<string>("projectId") || "";

      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Optional project id",
        value: defaultProjectId,
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const typeInput = await vscode.window.showInputBox({
        prompt: "Optional event type",
        placeHolder: "TASK_CREATED",
        ignoreFocusOut: true
      });

      if (typeInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      const eventType = typeInput.trim();
      const query = new URLSearchParams();
      if (projectId) {
        query.set("project_id", projectId);
      }
      if (eventType) {
        query.set("type", eventType);
      }

      const suffix = query.toString() ? `?${query.toString()}` : "";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/events/stream${suffix}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/events/stream${suffix}`;
      }

      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openAgents = vscode.commands.registerCommand(
    "cocopilot.openAgents",
    async () => {
      const targetUrl = buildUiUrl("/api/v2/agents");
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openLeases = vscode.commands.registerCommand(
    "cocopilot.openLeases",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL("/api/v2/leases", baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/leases`;
      }

      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openTasksApi = vscode.commands.registerCommand(
    "cocopilot.openTasksApi",
    async () => {
      const targetUrl = buildUiUrl("/api/v2/tasks");
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openTaskDetail = vscode.commands.registerCommand(
    "cocopilot.openTaskDetail",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedTaskId = encodeURIComponent(taskId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}`;
      }

      outputChannel.appendLine(`[VSIX] open task detail ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openRunDetail = vscode.commands.registerCommand(
    "cocopilot.openRunDetail",
    async () => {
      const runIdInput = await vscode.window.showInputBox({
        prompt: "Enter run id",
        placeHolder: "run_123",
        ignoreFocusOut: true
      });

      if (runIdInput === undefined) {
        return;
      }

      const runId = runIdInput.trim();
      if (!runId) {
        vscode.window.showWarningMessage("Run id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedRunId = encodeURIComponent(runId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/runs/${encodedRunId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/runs/${encodedRunId}`;
      }

      outputChannel.appendLine(`[VSIX] open run detail ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectDetail = vscode.commands.registerCommand(
    "cocopilot.openProjectDetail",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/projects/${encodedProjectId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}`;
      }

      outputChannel.appendLine(`[VSIX] open project detail ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectTree = vscode.commands.registerCommand(
    "cocopilot.openProjectTree",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/projects/${encodedProjectId}/tree`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/tree`;
      }

      outputChannel.appendLine(`[VSIX] open project tree ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectAudit = vscode.commands.registerCommand(
    "cocopilot.openProjectAudit",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/projects/${encodedProjectId}/audit`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/audit`;
      }

      outputChannel.appendLine(`[VSIX] open project audit ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectChanges = vscode.commands.registerCommand(
    "cocopilot.openProjectChanges",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const sinceInput = await vscode.window.showInputBox({
        prompt: "Optional since parameter",
        placeHolder: "2024-01-01T00:00:00Z",
        ignoreFocusOut: true
      });

      if (sinceInput === undefined) {
        return;
      }

      const since = sinceInput.trim();
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        const target = new URL(
          `/api/v2/projects/${encodedProjectId}/changes`,
          baseUrl
        );
        if (since) {
          target.searchParams.set("since", since);
        }
        targetUrl = target.toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        const suffix = since ? `?since=${encodeURIComponent(since)}` : "";
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/changes${suffix}`;
      }

      outputChannel.appendLine(`[VSIX] open project changes ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectMemory = vscode.commands.registerCommand(
    "cocopilot.openProjectMemory",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const scopeInput = await vscode.window.showInputBox({
        prompt: "Optional scope parameter",
        placeHolder: "tasks",
        ignoreFocusOut: true
      });

      if (scopeInput === undefined) {
        return;
      }

      const keyInput = await vscode.window.showInputBox({
        prompt: "Optional key parameter",
        placeHolder: "summary",
        ignoreFocusOut: true
      });

      if (keyInput === undefined) {
        return;
      }

      const qInput = await vscode.window.showInputBox({
        prompt: "Optional q parameter",
        placeHolder: "search terms",
        ignoreFocusOut: true
      });

      if (qInput === undefined) {
        return;
      }

      const scope = scopeInput.trim();
      const key = keyInput.trim();
      const q = qInput.trim();
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        const target = new URL(
          `/api/v2/projects/${encodedProjectId}/memory`,
          baseUrl
        );
        if (scope) {
          target.searchParams.set("scope", scope);
        }
        if (key) {
          target.searchParams.set("key", key);
        }
        if (q) {
          target.searchParams.set("q", q);
        }
        targetUrl = target.toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        const query = new URLSearchParams();
        if (scope) {
          query.set("scope", scope);
        }
        if (key) {
          query.set("key", key);
        }
        if (q) {
          query.set("q", q);
        }
        const suffix = query.toString() ? `?${query.toString()}` : "";
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/memory${suffix}`;
      }

      outputChannel.appendLine(`[VSIX] open project memory ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openProjectEventsReplay = vscode.commands.registerCommand(
    "cocopilot.openProjectEventsReplay",
    async () => {
      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Enter project id",
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const projectId = projectIdInput.trim();
      if (!projectId) {
        vscode.window.showWarningMessage("Project id is required.");
        return;
      }

      const sinceIdInput = await vscode.window.showInputBox({
        prompt: "Optional since_id parameter",
        placeHolder: "evt_123",
        ignoreFocusOut: true
      });

      if (sinceIdInput === undefined) {
        return;
      }

      const limitInput = await vscode.window.showInputBox({
        prompt: "Optional limit parameter",
        placeHolder: "100",
        ignoreFocusOut: true
      });

      if (limitInput === undefined) {
        return;
      }

      const sinceId = sinceIdInput.trim();
      const limit = limitInput.trim();
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        const target = new URL(
          `/api/v2/projects/${encodedProjectId}/events/replay`,
          baseUrl
        );
        if (sinceId) {
          target.searchParams.set("since_id", sinceId);
        }
        if (limit) {
          target.searchParams.set("limit", limit);
        }
        targetUrl = target.toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        const query = new URLSearchParams();
        if (sinceId) {
          query.set("since_id", sinceId);
        }
        if (limit) {
          query.set("limit", limit);
        }
        const suffix = query.toString() ? `?${query.toString()}` : "";
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/events/replay${suffix}`;
      }

      outputChannel.appendLine(`[VSIX] open project events replay ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openPolicies = vscode.commands.registerCommand(
    "cocopilot.openPolicies",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies`;
      }

      outputChannel.appendLine(`[VSIX] open policies ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const openPolicyDetail = vscode.commands.registerCommand(
    "cocopilot.openPolicyDetail",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const policyIdInput = await vscode.window.showInputBox({
        prompt: "Enter policy id",
        placeHolder: "policy_123",
        ignoreFocusOut: true
      });

      if (policyIdInput === undefined) {
        return;
      }

      const policyId = policyIdInput.trim();
      if (!policyId) {
        vscode.window.showWarningMessage("Policy id is required.");
        return;
      }

      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      const encodedPolicyId = encodeURIComponent(policyId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`;
      }

      outputChannel.appendLine(`[VSIX] open policy detail ${targetUrl}`);
      const target = vscode.Uri.parse(targetUrl);
      await vscode.env.openExternal(target);
    }
  );

  const listPolicies = vscode.commands.registerCommand(
    "cocopilot.listPolicies",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies`;
      }

      outputChannel.appendLine(`[VSIX] list policies GET ${targetUrl}`);

      try {
        const response = await fetch(targetUrl, { method: "GET", headers: getApiHeaders() });
        const responseText = await response.text();

        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to list policies (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] list policies failed (${response.status}): ${responseText}`
          );
          return;
        }

        let payload: unknown = undefined;
        try {
          payload = JSON.parse(responseText);
        } catch {
          payload = undefined;
        }

        const payloadObject = payload && typeof payload === "object" && !Array.isArray(payload)
          ? (payload as Record<string, unknown>)
          : undefined;
        const policies = Array.isArray(payloadObject?.policies)
          ? (payloadObject?.policies as unknown[])
          : Array.isArray(payload)
          ? (payload as unknown[])
          : [];

        if (policies.length === 0) {
          vscode.window.showInformationMessage("No policies returned.");
          outputChannel.appendLine(`[VSIX] list policies response: ${responseText}`);
          return;
        }

        const items = policies.map((item, index) => {
          if (item && typeof item === "object" && !Array.isArray(item)) {
            const policy = item as Record<string, unknown>;
            const id = policy.id ?? policy.policy_id ?? policy.key ?? policy.name;
            const titleValue = policy.name
              ?? policy.title
              ?? policy.rule
              ?? policy.type
              ?? (id ? `Policy ${id}` : `Policy ${index + 1}`);
            const statusValue = policy.status
              ?? policy.mode
              ?? policy.enforced
              ?? policy.enabled;
            const description = statusValue !== undefined
              ? String(statusValue)
              : undefined;
            return {
              label: String(titleValue),
              description
            } as vscode.QuickPickItem;
          }

          return {
            label: String(item ?? `Policy ${index + 1}`)
          } as vscode.QuickPickItem;
        });

        await vscode.window.showQuickPick(items, {
          title: "Cocopilot Policies",
          matchOnDescription: true,
          placeHolder: "Select a policy"
        });
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to list policies.");
        outputChannel.appendLine(`[VSIX] list policies error: ${detail}`);
      }
    }
  );

  const createPolicy = vscode.commands.registerCommand(
    "cocopilot.createPolicy",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const nameInput = await vscode.window.showInputBox({
        prompt: "Enter policy name",
        placeHolder: "Require approvals",
        ignoreFocusOut: true
      });

      if (nameInput === undefined) {
        return;
      }

      const name = nameInput.trim();
      if (!name) {
        vscode.window.showWarningMessage("Policy name is required.");
        return;
      }

      const descriptionInput = await vscode.window.showInputBox({
        prompt: "Enter policy description",
        placeHolder: "Require approvals before merge",
        ignoreFocusOut: true
      });

      if (descriptionInput === undefined) {
        return;
      }

      const description = descriptionInput.trim();
      if (!description) {
        vscode.window.showWarningMessage("Policy description is required.");
        return;
      }

      const rulesInput = await vscode.window.showInputBox({
        prompt: "Enter rules JSON",
        placeHolder: "{\"min_approvals\":2}",
        ignoreFocusOut: true
      });

      if (rulesInput === undefined) {
        return;
      }

      const rulesText = rulesInput.trim();
      if (!rulesText) {
        vscode.window.showWarningMessage("Rules JSON is required.");
        return;
      }

      let rules: unknown = undefined;
      try {
        rules = JSON.parse(rulesText);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showWarningMessage(`Invalid JSON rules: ${detail}`);
        return;
      }

      if (rules === null || (typeof rules !== "object" && !Array.isArray(rules))) {
        vscode.window.showWarningMessage("Rules must be a JSON object or array.");
        return;
      }

      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies`;
      }

      outputChannel.appendLine(
        `[VSIX] create policy POST ${targetUrl} project_id=${projectId}`
      );

      try {
        const response = await fetch(targetUrl, {
          method: "POST",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({
            name,
            description,
            rules
          })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to create policy (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] create policy failed (${response.status}): ${responseText}`
          );
          return;
        }

        let payload: Record<string, unknown> | undefined;
        try {
          payload = JSON.parse(responseText);
        } catch {
          payload = undefined;
        }

        const policyId = payload?.id ?? payload?.policy_id;
        if (policyId === undefined) {
          vscode.window.showInformationMessage("Created policy.");
        } else {
          vscode.window.showInformationMessage(`Created policy ${policyId}.`);
        }

        outputChannel.appendLine(`[VSIX] create policy response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to create policy.");
        outputChannel.appendLine(`[VSIX] create policy error: ${detail}`);
      }
    }
  );

  const updatePolicy = vscode.commands.registerCommand(
    "cocopilot.updatePolicy",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const policyIdInput = await vscode.window.showInputBox({
        prompt: "Enter policy id",
        placeHolder: "policy_123",
        ignoreFocusOut: true
      });

      if (policyIdInput === undefined) {
        return;
      }

      const policyId = policyIdInput.trim();
      if (!policyId) {
        vscode.window.showWarningMessage("Policy id is required.");
        return;
      }

      const nameInput = await vscode.window.showInputBox({
        prompt: "Enter policy name",
        placeHolder: "Updated policy",
        ignoreFocusOut: true
      });

      if (nameInput === undefined) {
        return;
      }

      const name = nameInput.trim();
      if (!name) {
        vscode.window.showWarningMessage("Policy name is required.");
        return;
      }

      const descriptionInput = await vscode.window.showInputBox({
        prompt: "Enter policy description",
        placeHolder: "Updated policy description",
        ignoreFocusOut: true
      });

      if (descriptionInput === undefined) {
        return;
      }

      const description = descriptionInput.trim();
      if (!description) {
        vscode.window.showWarningMessage("Policy description is required.");
        return;
      }

      const rulesInput = await vscode.window.showInputBox({
        prompt: "Enter rules JSON",
        placeHolder: "{\"min_approvals\":3}",
        ignoreFocusOut: true
      });

      if (rulesInput === undefined) {
        return;
      }

      const rulesText = rulesInput.trim();
      if (!rulesText) {
        vscode.window.showWarningMessage("Rules JSON is required.");
        return;
      }

      let rules: unknown = undefined;
      try {
        rules = JSON.parse(rulesText);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showWarningMessage(`Invalid JSON rules: ${detail}`);
        return;
      }

      if (rules === null || (typeof rules !== "object" && !Array.isArray(rules))) {
        vscode.window.showWarningMessage("Rules must be a JSON object or array.");
        return;
      }

      const enabledInput = await vscode.window.showInputBox({
        prompt: "Enable policy? (true/false)",
        placeHolder: "true",
        ignoreFocusOut: true
      });

      if (enabledInput === undefined) {
        return;
      }

      const enabledText = enabledInput.trim().toLowerCase();
      if (enabledText !== "true" && enabledText !== "false") {
        vscode.window.showWarningMessage("Enabled must be true or false.");
        return;
      }

      const enabled = enabledText === "true";
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      const encodedPolicyId = encodeURIComponent(policyId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`;
      }

      outputChannel.appendLine(
        `[VSIX] update policy PATCH ${targetUrl} project_id=${projectId} policy_id=${policyId}`
      );

      try {
        const response = await fetch(targetUrl, {
          method: "PATCH",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({
            name,
            description,
            rules,
            enabled
          })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to update policy (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] update policy failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Updated policy ${policyId}: ${trimmed}`
          : `Updated policy ${policyId}.`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] update policy response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to update policy.");
        outputChannel.appendLine(`[VSIX] update policy error: ${detail}`);
      }
    }
  );

  const togglePolicy = vscode.commands.registerCommand(
    "cocopilot.togglePolicy",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const policyIdInput = await vscode.window.showInputBox({
        prompt: "Enter policy id",
        placeHolder: "policy_123",
        ignoreFocusOut: true
      });

      if (policyIdInput === undefined) {
        return;
      }

      const policyId = policyIdInput.trim();
      if (!policyId) {
        vscode.window.showWarningMessage("Policy id is required.");
        return;
      }

      const enabledInput = await vscode.window.showInputBox({
        prompt: "Enable policy? (true/false)",
        placeHolder: "true",
        ignoreFocusOut: true
      });

      if (enabledInput === undefined) {
        return;
      }

      const enabledText = enabledInput.trim().toLowerCase();
      if (enabledText !== "true" && enabledText !== "false") {
        vscode.window.showWarningMessage("Enabled must be true or false.");
        return;
      }

      const enabled = enabledText === "true";
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      const encodedPolicyId = encodeURIComponent(policyId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`;
      }

      outputChannel.appendLine(
        `[VSIX] toggle policy PATCH ${targetUrl} project_id=${projectId} policy_id=${policyId}`
      );

      try {
        const response = await fetch(targetUrl, {
          method: "PATCH",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({
            enabled
          })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to toggle policy (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] toggle policy failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Updated policy ${policyId}: ${trimmed}`
          : `Updated policy ${policyId} (enabled=${enabled}).`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] toggle policy response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to toggle policy.");
        outputChannel.appendLine(`[VSIX] toggle policy error: ${detail}`);
      }
    }
  );

  const deletePolicy = vscode.commands.registerCommand(
    "cocopilot.deletePolicy",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const configuredProjectId = cocopilotConfig.get<string>("projectId") || "";
      let projectId = configuredProjectId.trim();

      if (!projectId) {
        const projectIdInput = await vscode.window.showInputBox({
          prompt: "Enter project id",
          placeHolder: "proj_123",
          ignoreFocusOut: true
        });

        if (projectIdInput === undefined) {
          return;
        }

        projectId = projectIdInput.trim();
        if (!projectId) {
          vscode.window.showWarningMessage("Project id is required.");
          return;
        }
      }

      const policyIdInput = await vscode.window.showInputBox({
        prompt: "Enter policy id",
        placeHolder: "policy_123",
        ignoreFocusOut: true
      });

      if (policyIdInput === undefined) {
        return;
      }

      const policyId = policyIdInput.trim();
      if (!policyId) {
        vscode.window.showWarningMessage("Policy id is required.");
        return;
      }

      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedProjectId = encodeURIComponent(projectId);
      const encodedPolicyId = encodeURIComponent(policyId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(
          `/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`,
          baseUrl
        ).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/policies/${encodedPolicyId}`;
      }

      outputChannel.appendLine(
        `[VSIX] delete policy DELETE ${targetUrl} project_id=${projectId} policy_id=${policyId}`
      );

      try {
        const response = await fetch(targetUrl, { method: "DELETE", headers: getApiHeaders() });
        const responseText = await response.text();

        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to delete policy (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] delete policy failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Deleted policy ${policyId}: ${trimmed}`
          : `Deleted policy ${policyId}.`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] delete policy response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to delete policy.");
        outputChannel.appendLine(`[VSIX] delete policy error: ${detail}`);
      }
    }
  );

  const createTask = vscode.commands.registerCommand(
    "cocopilot.createTask",
    async () => {
      const instructionsInput = await vscode.window.showInputBox({
        prompt: "Enter task instructions",
        placeHolder: "Describe the task to create",
        ignoreFocusOut: true
      });

      if (instructionsInput === undefined) {
        return;
      }

      const instructions = instructionsInput.trim();
      if (!instructions) {
        vscode.window.showWarningMessage("Task instructions are required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const projectId = cocopilotConfig.get<string>("projectId") || "";
      if (!projectId.trim()) {
        vscode.window.showWarningMessage("Set a project ID (Cocopilot: Set Project ID) before creating tasks.");
        return;
      }

      const encodedProjectId = encodeURIComponent(projectId.trim());
      let targetUrl = "";
      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/projects/${encodedProjectId}/tasks`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/tasks`;
      }

      outputChannel.appendLine(`[VSIX] create task POST ${targetUrl}`);

      try {
        const response = await fetch(targetUrl, {
          method: "POST",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({ instructions })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to create task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] create task failed (${response.status}): ${responseText}`
          );
          return;
        }

        let payload: { id?: number | string } | undefined;
        try {
          payload = JSON.parse(responseText);
        } catch {
          payload = undefined;
        }

        const taskId = payload?.id;
        if (taskId === undefined || taskId === null) {
          vscode.window.showInformationMessage(
            "Task created, but no task id was returned."
          );
          outputChannel.appendLine(
            `[VSIX] create task response: ${responseText}`
          );
          return;
        }

        vscode.window.showInformationMessage(`Created task ${taskId}.`);
        outputChannel.appendLine(`[VSIX] created task ${taskId}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to create task.");
        outputChannel.appendLine(`[VSIX] create task error: ${detail}`);
      }
    }
  );

  const saveTask = vscode.commands.registerCommand(
    "cocopilot.saveTask",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const messageInput = await vscode.window.showInputBox({
        prompt: "Enter completion message (what changed)",
        placeHolder: "Summary of changes or results",
        ignoreFocusOut: true
      });

      if (messageInput === undefined) {
        return;
      }

      const message = messageInput.trim();
      if (!message) {
        vscode.window.showWarningMessage("Message is required.");
        return;
      }

      const filesInput = await vscode.window.showInputBox({
        prompt: "Files touched (comma-separated, optional)",
        placeHolder: "src/main.go, README.md",
        ignoreFocusOut: true
      });

      const filesTouched = filesInput
        ? filesInput.split(",").map(f => f.trim()).filter(Boolean)
        : undefined;

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedTaskId = encodeURIComponent(taskId);

      let targetUrl = "";
      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}/complete`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}/complete`;
      }

      const payload: Record<string, unknown> = {
        output: message,
        summary: {
          what_changed: message,
          ...(filesTouched?.length ? { files_touched: filesTouched } : {})
        }
      };

      outputChannel.appendLine(`[VSIX] complete task POST ${targetUrl} task_id=${taskId}`);

      try {
        const response = await fetch(targetUrl, {
          method: "POST",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify(payload)
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to complete task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] complete task failed (${response.status}): ${responseText}`
          );
          return;
        }

        vscode.window.showInformationMessage(`Completed task ${taskId}.`);
        outputChannel.appendLine(`[VSIX] complete task response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to complete task.");
        outputChannel.appendLine(`[VSIX] complete task error: ${detail}`);
      }
    }
  );

  const failTask = vscode.commands.registerCommand(
    "cocopilot.failTask",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id to fail",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const errorInput = await vscode.window.showInputBox({
        prompt: "Enter error message",
        placeHolder: "Reason for failure",
        ignoreFocusOut: true
      });

      if (errorInput === undefined) {
        return;
      }

      const errorMessage = errorInput.trim();
      if (!errorMessage) {
        vscode.window.showWarningMessage("Error message is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedTaskId = encodeURIComponent(taskId);

      let targetUrl = "";
      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}/complete`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}/complete`;
      }

      outputChannel.appendLine(`[VSIX] fail task POST ${targetUrl} task_id=${taskId}`);

      try {
        const response = await fetch(targetUrl, {
          method: "POST",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({
            status: "FAILED",
            error: errorMessage,
            output: errorMessage
          })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to fail task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] fail task failed (${response.status}): ${responseText}`
          );
          return;
        }

        vscode.window.showInformationMessage(`Failed task ${taskId}.`);
        outputChannel.appendLine(`[VSIX] fail task response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to fail task.");
        outputChannel.appendLine(`[VSIX] fail task error: ${detail}`);
      }
    }
  );

  const updateStatus = vscode.commands.registerCommand(
    "cocopilot.updateStatus",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const statusInput = await vscode.window.showInputBox({
        prompt: "Enter status",
        placeHolder: "RUNNING",
        ignoreFocusOut: true
      });

      if (statusInput === undefined) {
        return;
      }

      const status = statusInput.trim();
      if (!status) {
        vscode.window.showWarningMessage("Status is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedTaskId = encodeURIComponent(taskId);

      let targetUrl = "";
      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}`;
      }

      outputChannel.appendLine(
        `[VSIX] update status PATCH ${targetUrl} task_id=${taskId} status=${status}`
      );

      try {
        const response = await fetch(targetUrl, {
          method: "PATCH",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify({ status })
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to update status (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] update status failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Updated task ${taskId} status to ${status}: ${trimmed}`
          : `Updated task ${taskId} status to ${status}.`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] update status response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to update status.");
        outputChannel.appendLine(`[VSIX] update status error: ${detail}`);
      }
    }
  );

  const updateTask = vscode.commands.registerCommand(
    "cocopilot.updateTask",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const payloadInput = await vscode.window.showInputBox({
        prompt: "Enter JSON payload",
        placeHolder: "{\"title\":\"...\",\"instructions\":\"...\",\"status\":\"RUNNING\",\"type\":\"BUILD\",\"tags\":[\"api\"]}",
        ignoreFocusOut: true
      });

      if (payloadInput === undefined) {
        return;
      }

      const payloadText = payloadInput.trim();
      if (!payloadText) {
        vscode.window.showWarningMessage("Payload is required.");
        return;
      }

      let payload: Record<string, unknown> | undefined;
      try {
        const parsed = JSON.parse(payloadText);
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          throw new Error("Payload must be a JSON object.");
        }

        const allowedKeys = new Set([
          "title",
          "instructions",
          "status",
          "type",
          "tags"
        ]);

        payload = Object.entries(parsed).reduce<Record<string, unknown>>(
          (acc, [key, value]) => {
            if (allowedKeys.has(key)) {
              acc[key] = value;
            }
            return acc;
          },
          {}
        );
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showWarningMessage(`Invalid JSON payload: ${detail}`);
        return;
      }

      if (!payload || Object.keys(payload).length === 0) {
        vscode.window.showWarningMessage("Payload must include at least one field.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const encodedTaskId = encodeURIComponent(taskId);
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}`;
      }

      outputChannel.appendLine(`[VSIX] update task PATCH ${targetUrl} task_id=${taskId}`);

      try {
        const response = await fetch(targetUrl, {
          method: "PATCH",
          headers: getApiHeaders({
            "Content-Type": "application/json"
          }),
          body: JSON.stringify(payload)
        });

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to update task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] update task failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Updated task ${taskId}: ${trimmed}`
          : `Updated task ${taskId}.`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] update task response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to update task.");
        outputChannel.appendLine(`[VSIX] update task error: ${detail}`);
      }
    }
  );

  const deleteTask = vscode.commands.registerCommand(
    "cocopilot.deleteTask",
    async () => {
      const taskIdInput = await vscode.window.showInputBox({
        prompt: "Enter task id",
        placeHolder: "123",
        ignoreFocusOut: true
      });

      if (taskIdInput === undefined) {
        return;
      }

      const taskId = taskIdInput.trim();
      if (!taskId) {
        vscode.window.showWarningMessage("Task id is required.");
        return;
      }

      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      let targetUrl = "";
      const encodedTaskId = encodeURIComponent(taskId);

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks/${encodedTaskId}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks/${encodedTaskId}`;
      }

      outputChannel.appendLine(`[VSIX] delete task DELETE ${targetUrl}`);

      try {
        const response = await fetch(targetUrl, { method: "DELETE", headers: getApiHeaders() });
        const responseText = await response.text();

        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to delete task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] delete task failed (${response.status}): ${responseText}`
          );
          return;
        }

        const trimmed = responseText.trim();
        const messageText = trimmed
          ? `Deleted task ${taskId}: ${trimmed}`
          : `Deleted task ${taskId}.`;
        vscode.window.showInformationMessage(messageText);
        outputChannel.appendLine(`[VSIX] delete task response: ${responseText}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to delete task.");
        outputChannel.appendLine(`[VSIX] delete task error: ${detail}`);
      }
    }
  );

  const listTasks = vscode.commands.registerCommand(
    "cocopilot.listTasks",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const defaultProjectId = cocopilotConfig.get<string>("projectId") || "";

      const statusInput = await vscode.window.showInputBox({
        prompt: "Optional status filter",
        placeHolder: "RUNNING",
        ignoreFocusOut: true
      });

      if (statusInput === undefined) {
        return;
      }

      const projectIdInput = await vscode.window.showInputBox({
        prompt: "Optional project id filter",
        value: defaultProjectId,
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (projectIdInput === undefined) {
        return;
      }

      const status = statusInput.trim();
      const projectId = projectIdInput.trim();
      const query = new URLSearchParams();
      if (status) {
        query.set("status", status);
      }
      if (projectId) {
        query.set("project_id", projectId);
      }

      const suffix = query.toString() ? `?${query.toString()}` : "";
      let targetUrl = "";

      try {
        const baseUrl = new URL(apiBase);
        targetUrl = new URL(`/api/v2/tasks${suffix}`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        targetUrl = `${trimmed}/api/v2/tasks${suffix}`;
      }

      outputChannel.appendLine(`[VSIX] list tasks GET ${targetUrl}`);

      try {
        const response = await fetch(targetUrl, { method: "GET", headers: getApiHeaders() });
        const responseText = await response.text();

        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to list tasks (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] list tasks failed (${response.status}): ${responseText}`
          );
          return;
        }

        let payload: { tasks?: unknown[]; total?: number } | undefined;
        try {
          payload = JSON.parse(responseText);
        } catch {
          payload = undefined;
        }

        const tasks = Array.isArray(payload?.tasks) ? payload?.tasks : [];
        if (tasks.length === 0) {
          vscode.window.showInformationMessage("No tasks returned.");
          outputChannel.appendLine(`[VSIX] list tasks response: ${responseText}`);
          return;
        }

        const items = tasks.map((item) => {
          const task = (item ?? {}) as Record<string, unknown>;
          const id = task.id ?? task.task_id;
          const titleValue = task.title ?? task.instructions ?? (id ? `Task ${id}` : "Untitled task");
          const statusValue = task.status_v2 ?? task.status ?? task.status_v1 ?? "unknown";
          return {
            label: String(titleValue),
            description: String(statusValue)
          } as vscode.QuickPickItem;
        });

        await vscode.window.showQuickPick(items, {
          title: "Cocopilot Tasks",
          matchOnDescription: true,
          placeHolder: "Select a task"
        });
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to list tasks.");
        outputChannel.appendLine(`[VSIX] list tasks error: ${detail}`);
      }
    }
  );

  const claimTask = vscode.commands.registerCommand(
    "cocopilot.claimTask",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const apiBase = cocopilotConfig.get<string>("apiBase") || "http://localhost:8080";
      const projectId = cocopilotConfig.get<string>("projectId") || "";
      const agentId = cocopilotConfig.get<string>("agentId") || "vsix-agent";

      if (!projectId.trim()) {
        vscode.window.showWarningMessage("Set a project ID (Cocopilot: Set Project ID) before claiming tasks.");
        return;
      }

      const encodedProjectId = encodeURIComponent(projectId.trim());

      // Use claim-next endpoint to claim the next available task.
      let claimNextUrl = "";
      try {
        const baseUrl = new URL(apiBase);
        claimNextUrl = new URL(`/api/v2/projects/${encodedProjectId}/tasks/claim-next`, baseUrl).toString();
      } catch {
        const trimmed = apiBase.replace(/\/+$/, "");
        claimNextUrl = `${trimmed}/api/v2/projects/${encodedProjectId}/tasks/claim-next`;
      }

      outputChannel.appendLine(`[VSIX] claim task POST ${claimNextUrl}`);

      try {
        const response = await fetch(claimNextUrl, {
          method: "POST",
          headers: getApiHeaders({
            "Content-Type": "application/json",
            "X-Agent-ID": agentId
          }),
          body: JSON.stringify({ agent_id: agentId })
        });

        if (response.status === 204) {
          vscode.window.showInformationMessage("No tasks available.");
          outputChannel.appendLine("[VSIX] claim task: no tasks available (204)");
          return;
        }

        const responseText = await response.text();
        if (!response.ok) {
          vscode.window.showErrorMessage(
            `Failed to claim task (${response.status}).`
          );
          outputChannel.appendLine(
            `[VSIX] claim task failed (${response.status}): ${responseText}`
          );
          return;
        }

        let envelope: {
          task?: { id?: number; instructions?: string; title?: string; type?: string; priority?: number; tags?: string[] };
          lease?: { id?: string; expires_at?: string };
          run?: { id?: string };
          project?: { id?: string; name?: string };
          completion_contract?: {
            required_fields?: string[];
            expected_outputs?: string[];
            notes?: string;
          };
          context?: {
            memories?: { key?: string; scope?: string; value?: Record<string, unknown> }[];
            policies?: { name?: string; enabled?: boolean }[];
            dependencies?: { task_id?: number; depends_on_task_id?: number }[];
            recent_run_summaries?: { id?: string; status?: string; error_message?: string }[];
            repo_files?: string[];
          };
        } | undefined;
        try {
          envelope = JSON.parse(responseText);
        } catch {
          envelope = undefined;
        }

        const task = envelope?.task;
        const contract = envelope?.completion_contract;
        const ctx = envelope?.context;

        const instructions = task?.instructions ?? "";
        const title = task?.title;
        const taskId = task?.id;

        const sections: string[] = [];

        // Task header
        if (title) {
          sections.push(`# ${title} (Task ${taskId})`);
        } else {
          sections.push(`# Task ${taskId}`);
        }

        if (task?.type) {
          sections.push(`Type: ${task.type}  Priority: ${task.priority ?? "default"}`);
        }
        if (task?.tags?.length) {
          sections.push(`Tags: ${task.tags.join(", ")}`);
        }

        // Lease & run
        if (envelope?.lease) {
          sections.push(`Lease: ${envelope.lease.id} (expires ${envelope.lease.expires_at})`);
        }
        if (envelope?.run) {
          sections.push(`Run: ${envelope.run.id}`);
        }
        if (envelope?.project) {
          sections.push(`Project: ${envelope.project.name ?? envelope.project.id}`);
        }

        sections.push("");
        sections.push("## Instructions");
        sections.push(instructions);

        // Completion contract
        if (contract) {
          sections.push("");
          sections.push("## Completion Contract");
          if (contract.required_fields?.length) {
            sections.push(`Required fields: ${contract.required_fields.join(", ")}`);
          }
          if (contract.expected_outputs?.length) {
            sections.push(`Expected outputs: ${contract.expected_outputs.join(", ")}`);
          }
          if (contract.notes) {
            sections.push(`Notes: ${contract.notes}`);
          }
        }

        // Context
        if (ctx) {
          if (ctx.dependencies?.length) {
            sections.push("");
            sections.push("## Dependencies");
            for (const dep of ctx.dependencies) {
              sections.push(`- Task ${dep.task_id} depends on ${dep.depends_on_task_id}`);
            }
          }
          if (ctx.recent_run_summaries?.length) {
            sections.push("");
            sections.push("## Recent Runs");
            for (const run of ctx.recent_run_summaries) {
              const err = run.error_message ? ` — ${run.error_message}` : "";
              sections.push(`- ${run.id}: ${run.status}${err}`);
            }
          }
          if (ctx.memories?.length) {
            sections.push("");
            sections.push("## Memories");
            for (const mem of ctx.memories) {
              sections.push(`- [${mem.scope}] ${mem.key}`);
            }
          }
          if (ctx.policies?.length) {
            sections.push("");
            sections.push("## Policies");
            for (const pol of ctx.policies) {
              sections.push(`- ${pol.name} (${pol.enabled ? "enabled" : "disabled"})`);
            }
          }
          if (ctx.repo_files?.length) {
            sections.push("");
            sections.push("## Repo Files");
            sections.push(ctx.repo_files.slice(0, 20).join(", "));
          }
        }

        const displayContent = sections.join("\n");

        const doc = await vscode.workspace.openTextDocument({
          content: displayContent,
          language: "markdown"
        });
        await vscode.window.showTextDocument(doc, { preview: false });
        const msg = title ? `Claimed: ${title} (task ${taskId})` : `Claimed task ${taskId}.`;
        vscode.window.showInformationMessage(msg);

        outputChannel.appendLine(`[VSIX] claim task response: task=${taskId} title=${title ?? "(none)"}`);
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage("Failed to claim task.");
        outputChannel.appendLine(`[VSIX] claim task error: ${detail}`);
      }
    }
  );

  const openSettings = vscode.commands.registerCommand(
    "cocopilot.openSettings",
    async () => {
      await vscode.commands.executeCommand("workbench.action.openSettings", "cocopilot");
    }
  );

  const showQuickStart = vscode.commands.registerCommand(
    "cocopilot.showQuickStart",
    async () => {
      const selection = await vscode.window.showInformationMessage(
        "Cocopilot quick start: configure MCP, start the MCP server, or open the dashboard.",
        "Configure MCP",
        "Start MCP",
        "Open Dashboard"
      );

      if (selection === "Configure MCP") {
        await vscode.commands.executeCommand("cocopilot.configureMcp");
      } else if (selection === "Start MCP") {
        await vscode.commands.executeCommand("cocopilot.startMcpServer");
      } else if (selection === "Open Dashboard") {
        await vscode.commands.executeCommand("cocopilot.openDashboard");
      }
    }
  );

  const setProjectId = vscode.commands.registerCommand(
    "cocopilot.setProjectId",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const current = cocopilotConfig.get<string>("projectId") || "";
      const value = await vscode.window.showInputBox({
        prompt: "Enter Cocopilot project ID",
        value: current,
        placeHolder: "proj_123",
        ignoreFocusOut: true
      });

      if (value === undefined) {
        return;
      }

      const trimmed = value.trim();
      const target = vscode.workspace.workspaceFolders?.length
        ? vscode.ConfigurationTarget.Workspace
        : vscode.ConfigurationTarget.Global;
      await cocopilotConfig.update("projectId", trimmed, target);
      updateStatusBar();

      if (trimmed) {
        vscode.window.showInformationMessage(
          `Cocopilot project ID set to ${trimmed}.`
        );
      } else {
        vscode.window.showInformationMessage("Cocopilot project ID cleared.");
      }
    }
  );

  const setApiBase = vscode.commands.registerCommand(
    "cocopilot.setApiBase",
    async () => {
      const cocopilotConfig = vscode.workspace.getConfiguration("cocopilot");
      const current = cocopilotConfig.get<string>("apiBase") || "";
      const value = await vscode.window.showInputBox({
        prompt: "Enter Cocopilot API base URL",
        value: current,
        placeHolder: "http://localhost:8080",
        ignoreFocusOut: true
      });

      if (value === undefined) {
        return;
      }

      const trimmed = value.trim();
      const target = vscode.workspace.workspaceFolders?.length
        ? vscode.ConfigurationTarget.Workspace
        : vscode.ConfigurationTarget.Global;
      await cocopilotConfig.update("apiBase", trimmed, target);

      if (trimmed) {
        vscode.window.showInformationMessage(
          `Cocopilot API base set to ${trimmed}.`
        );
      } else {
        vscode.window.showInformationMessage("Cocopilot API base cleared.");
      }
    }
  );

  const openApiDocs = vscode.commands.registerCommand(
    "cocopilot.openApiDocs",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const docsUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "README.md"
      );
      const doc = await vscode.workspace.openTextDocument(docsUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openOpenApiSpec = vscode.commands.registerCommand(
    "cocopilot.openOpenApiSpec",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const specUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "openapi-v2.yaml"
      );
      const doc = await vscode.workspace.openTextDocument(specUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openApiSummary = vscode.commands.registerCommand(
    "cocopilot.openApiSummary",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const summaryUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "v2-summary.md"
      );
      const doc = await vscode.workspace.openTextDocument(summaryUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openApiDesign = vscode.commands.registerCommand(
    "cocopilot.openApiDesign",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const designUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "v2-design.md"
      );
      const doc = await vscode.workspace.openTextDocument(designUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openApiCompatibility = vscode.commands.registerCommand(
    "cocopilot.openApiCompatibility",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const compatUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "v2-compatibility.md"
      );
      const doc = await vscode.workspace.openTextDocument(compatUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openApiRoadmap = vscode.commands.registerCommand(
    "cocopilot.openApiRoadmap",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const roadmapUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "api",
        "v2-roadmap.md"
      );
      const doc = await vscode.workspace.openTextDocument(roadmapUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openMigrationsDocs = vscode.commands.registerCommand(
    "cocopilot.openMigrationsDocs",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const docsUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "MIGRATIONS.md"
      );
      const doc = await vscode.workspace.openTextDocument(docsUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openRoadmap = vscode.commands.registerCommand(
    "cocopilot.openRoadmap",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const roadmapUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "ROADMAP.md"
      );
      const doc = await vscode.workspace.openTextDocument(roadmapUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openKbOverview = vscode.commands.registerCommand(
    "cocopilot.openKbOverview",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const overviewUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "ai",
        "kb",
        "00-overview.md"
      );
      const doc = await vscode.workspace.openTextDocument(overviewUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateDocs = vscode.commands.registerCommand(
    "cocopilot.openStateDocs",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const overviewUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "overview.md"
      );
      const currentUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "current.md"
      );

      let docUri = overviewUri;
      try {
        await vscode.workspace.fs.stat(overviewUri);
      } catch {
        docUri = currentUri;
      }

      const doc = await vscode.workspace.openTextDocument(docUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateArchitecture = vscode.commands.registerCommand(
    "cocopilot.openStateArchitecture",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const architectureUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "architecture.md"
      );
      const doc = await vscode.workspace.openTextDocument(architectureUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateDecisions = vscode.commands.registerCommand(
    "cocopilot.openStateDecisions",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const decisionsUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "decisions.md"
      );
      const doc = await vscode.workspace.openTextDocument(decisionsUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateCurrent = vscode.commands.registerCommand(
    "cocopilot.openStateCurrent",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const currentUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "current.md"
      );
      const doc = await vscode.workspace.openTextDocument(currentUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateNext = vscode.commands.registerCommand(
    "cocopilot.openStateNext",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const nextUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "next.md"
      );
      const doc = await vscode.workspace.openTextDocument(nextUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openStateRisks = vscode.commands.registerCommand(
    "cocopilot.openStateRisks",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const risksUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "docs",
        "state",
        "risks.md"
      );
      const doc = await vscode.workspace.openTextDocument(risksUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openMcpReadme = vscode.commands.registerCommand(
    "cocopilot.openMcpReadme",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const readmeUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "tools",
        "cocopilot-mcp",
        "README.md"
      );
      const doc = await vscode.workspace.openTextDocument(readmeUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const openMcpTools = vscode.commands.registerCommand(
    "cocopilot.openMcpTools",
    async () => {
      const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
      if (!workspaceFolder) {
        vscode.window.showErrorMessage("Open a workspace folder first.");
        return;
      }

      const toolsUri = vscode.Uri.joinPath(
        workspaceFolder.uri,
        "tools",
        "cocopilot-mcp",
        "tools.json"
      );
      const doc = await vscode.workspace.openTextDocument(toolsUri);
      await vscode.window.showTextDocument(doc, { preview: false });
    }
  );

  const showLogs = vscode.commands.registerCommand(
    "cocopilot.showLogs",
    () => {
      outputChannel.show(true);
    }
  );

  const configListener = vscode.workspace.onDidChangeConfiguration((event) => {
    if (event.affectsConfiguration("cocopilot.projectId")) {
      updateStatusBar();
    }
  });

  const terminalCloseListener = vscode.window.onDidCloseTerminal((terminal) => {
    if (mcpTerminal && terminal === mcpTerminal) {
      mcpTerminal = undefined;
      updateMcpStatusBar(false);
    }
  });

  updateStatusBar();
  updateMcpStatusBar(!!mcpTerminal);

  const autoStartMcp = vscode.workspace
    .getConfiguration("cocopilot")
    .get<boolean>("autoStartMcpServer");
  if (autoStartMcp) {
    logMcp("auto-start: enabled");
    void startMcpServerNow(false).catch((err) => {
      logMcpError("auto-start: failed", err);
    });
  }

  context.subscriptions.push(disposable);
  context.subscriptions.push(configureMcp);
  context.subscriptions.push(openMcpConfig);
  context.subscriptions.push(startMcpServer);
  context.subscriptions.push(stopMcpServer);
  context.subscriptions.push(openDashboard);
  context.subscriptions.push(openTasksBoard);
  context.subscriptions.push(openHealth);
  context.subscriptions.push(openConfig);
  context.subscriptions.push(openVersion);
  context.subscriptions.push(openEvents);
  context.subscriptions.push(openEventsStream);
  context.subscriptions.push(openAgents);
  context.subscriptions.push(openLeases);
  context.subscriptions.push(openTasksApi);
  context.subscriptions.push(openTaskDetail);
  context.subscriptions.push(openRunDetail);
  context.subscriptions.push(openProjectDetail);
  context.subscriptions.push(openProjectTree);
  context.subscriptions.push(openProjectAudit);
  context.subscriptions.push(openProjectChanges);
  context.subscriptions.push(openProjectMemory);
  context.subscriptions.push(openProjectEventsReplay);
  context.subscriptions.push(openPolicies);
  context.subscriptions.push(openPolicyDetail);
  context.subscriptions.push(listPolicies);
  context.subscriptions.push(createPolicy);
  context.subscriptions.push(updatePolicy);
  context.subscriptions.push(togglePolicy);
  context.subscriptions.push(deletePolicy);
  context.subscriptions.push(createTask);
  context.subscriptions.push(saveTask);
  context.subscriptions.push(failTask);
  context.subscriptions.push(updateStatus);
  context.subscriptions.push(updateTask);
  context.subscriptions.push(deleteTask);
  context.subscriptions.push(listTasks);
  context.subscriptions.push(claimTask);
  context.subscriptions.push(openSettings);
  context.subscriptions.push(showQuickStart);
  context.subscriptions.push(setProjectId);
  context.subscriptions.push(setApiBase);
  context.subscriptions.push(openApiDocs);
  context.subscriptions.push(openOpenApiSpec);
  context.subscriptions.push(openApiSummary);
  context.subscriptions.push(openApiDesign);
  context.subscriptions.push(openApiCompatibility);
  context.subscriptions.push(openApiRoadmap);
  context.subscriptions.push(openMigrationsDocs);
  context.subscriptions.push(openRoadmap);
  context.subscriptions.push(openKbOverview);
  context.subscriptions.push(openStateDocs);
  context.subscriptions.push(openStateArchitecture);
  context.subscriptions.push(openStateDecisions);
  context.subscriptions.push(openStateCurrent);
  context.subscriptions.push(openStateNext);
  context.subscriptions.push(openStateRisks);
  context.subscriptions.push(openMcpReadme);
  context.subscriptions.push(openMcpTools);
  context.subscriptions.push(showLogs);

  // Alias commands for checklist compatibility
  const startServer = vscode.commands.registerCommand(
    "cocopilot.startServer",
    async () => {
      await vscode.commands.executeCommand("cocopilot.startMcpServer");
    }
  );
  const stopServer = vscode.commands.registerCommand(
    "cocopilot.stopServer",
    async () => {
      await vscode.commands.executeCommand("cocopilot.stopMcpServer");
    }
  );
  const showTasks = vscode.commands.registerCommand(
    "cocopilot.showTasks",
    async () => {
      await vscode.commands.executeCommand("cocopilot.openTasksBoard");
    }
  );
  context.subscriptions.push(startServer);
  context.subscriptions.push(stopServer);
  context.subscriptions.push(showTasks);

  context.subscriptions.push(configListener);
  context.subscriptions.push(terminalCloseListener);
  context.subscriptions.push(statusBar);
  context.subscriptions.push(mcpStatusBar);
  context.subscriptions.push(outputChannel);
}

export function deactivate() {
  console.log("cocopilot-vsix deactivated");
}
