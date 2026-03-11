package server

const htmlTemplate = `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Cocopilot - Work Queue</title>
        <script defer src="/static/alpine-collapse.min.js"></script>
        <script defer src="/static/alpine.min.js"></script>
        <style>
            * {
                box-sizing: border-box;
                margin: 0;
                padding: 0;
            }

            :root {
                --vscode-bg: #1e1e1e;
                --vscode-sidebar: #252526;
                --vscode-input-bg: #3c3c3c;
                --vscode-border: #3c3c3c;
                --vscode-text: #cccccc;
                --vscode-text-muted: #858585;
                --vscode-accent: #0078d4;
                --vscode-accent-hover: #1c8ae8;
                --vscode-success: #89d185;
                --vscode-warning: #cca700;
                --vscode-info: #3794ff;
                --vscode-error: #f48771;
                --vscode-dropdown-bg: #313131;
                --vscode-charts-green: #388a34;
                --vscode-charts-red: #c74e39;
                --vscode-charts-orange: #d18616;
                --vscode-charts-blue: #2670b8;
            }

            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                font-size: 13px;
                background: var(--vscode-bg);
                color: var(--vscode-text);
                height: 100vh;
                display: flex;
                flex-direction: column;
                overflow: hidden;
            }

            /* Header */
            .header {
                background: var(--vscode-sidebar);
                border-bottom: 1px solid var(--vscode-border);
                padding: 12px 20px;
                display: flex;
                align-items: center;
                gap: 12px;
                flex-shrink: 0;
            }

            .header-icon { color: var(--vscode-accent); }
            .header-title { font-size: 14px; font-weight: 600; }

            .header-nav {
                display: flex;
                align-items: center;
                gap: 10px;
                font-size: 12px;
            }

            .header-nav a {
                color: var(--vscode-text-muted);
                text-decoration: none;
                padding: 4px 6px;
                border-radius: 4px;
                transition: all 0.15s;
            }

            .header-nav a:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }
            .header-nav a.active {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
                font-weight: 600;
            }
            .nav-more {
                position: relative;
            }
            .nav-more-btn {
                color: var(--vscode-text-muted);
                background: none;
                border: none;
                cursor: pointer;
                font-size: 12px;
                padding: 4px 6px;
                border-radius: 4px;
                transition: all 0.15s;
            }
            .nav-more-btn:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }
            .nav-more-dropdown {
                position: absolute;
                top: 100%;
                left: 0;
                background: var(--vscode-dropdown-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                box-shadow: 0 4px 12px rgba(0,0,0,0.2);
                min-width: 160px;
                z-index: 1000;
                padding: 4px 0;
            }
            .nav-more-dropdown a {
                display: block;
                padding: 6px 12px;
                color: var(--vscode-text-muted);
                text-decoration: none;
                font-size: 12px;
            }
            .nav-more-dropdown a:hover {
                color: var(--vscode-text);
                background: var(--vscode-input-bg);
            }

            .header-actions {
                margin-left: auto;
                display: flex;
                align-items: center;
                gap: 12px;
            }

            .header-btn {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 6px 12px;
                font-size: 12px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 6px;
                transition: all 0.15s;
            }

            .header-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .workdir-input {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .workdir-input label {
                font-size: 12px;
                color: var(--vscode-text-muted);
            }

            .workdir-input input {
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 4px;
                color: var(--vscode-text);
                padding: 6px 10px;
                font-size: 12px;
                width: 250px;
                outline: none;
            }

            .workdir-input input:focus {
                border-color: var(--vscode-accent);
            }

            /* Kanban Board */
            .kanban-board {
                flex: 1;
                display: grid;
                grid-template-columns: repeat(3, 1fr);
                gap: 16px;
                padding: 20px;
                overflow: hidden;
                min-height: 0;
            }

            .kanban-column {
                background: var(--vscode-sidebar);
                border-radius: 8px;
                display: flex;
                flex-direction: column;
                overflow: hidden;
                border: 1px solid var(--vscode-border);
                transition: border-color 0.2s;
                min-height: 0;
            }

            .kanban-column.drag-over {
                border-color: var(--vscode-accent);
                box-shadow: 0 0 0 2px rgba(0, 120, 212, 0.3);
            }

            .column-header {
                padding: 12px 16px;
                border-bottom: 1px solid var(--vscode-border);
                display: flex;
                align-items: center;
                gap: 8px;
                flex-shrink: 0;
            }

            .column-title { font-weight: 600; font-size: 13px; }

            .column-count {
                background: var(--vscode-input-bg);
                padding: 2px 8px;
                border-radius: 10px;
                font-size: 11px;
                color: var(--vscode-text-muted);
                transition: all 0.3s;
            }

            .column-icon { width: 16px; height: 16px; }
            .col-todo .column-icon { color: var(--vscode-warning); }
            .col-progress .column-icon { color: var(--vscode-info); }
            .col-done .column-icon { color: var(--vscode-success); }

            .column-body {
                flex: 1;
                overflow-y: auto;
                padding: 12px;
                display: flex;
                flex-direction: column;
                gap: 10px;
                min-height: 0;
            }

            .column-body::-webkit-scrollbar { width: 8px; }
            .column-body::-webkit-scrollbar-track { background: transparent; }
            .column-body::-webkit-scrollbar-thumb { background: var(--vscode-border); border-radius: 4px; }

            /* Task Cards */
            .task-card {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                cursor: grab;
                transition: all 0.2s ease;
                overflow: hidden;
                flex-shrink: 0;
            }

            .task-card:hover {
                border-color: var(--vscode-accent);
                box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
            }

            .task-card.dragging {
                opacity: 0.5;
                cursor: grabbing;
                transform: scale(0.98);
            }

            .task-card.task-blocked {
                border-left: 3px solid var(--vscode-error);
                opacity: 0.7;
            }

            .card-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 10px 12px;
                cursor: pointer;
                user-select: none;
            }

            .card-header:hover { background: rgba(255,255,255,0.03); }

            .card-left {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-id {
                font-weight: 600;
                color: var(--vscode-accent);
                font-size: 12px;
            }

            .card-preview {
                font-size: 11px;
                color: var(--vscode-text-muted);
                max-width: 150px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            .card-right {
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .card-time {
                font-size: 10px;
                color: var(--vscode-text-muted);
            }

            .delete-btn {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 2px;
                border-radius: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
                transition: all 0.15s;
            }

            .delete-btn:hover {
                background: rgba(255, 0, 0, 0.2);
                color: #ff6b6b;
            }

            .expand-icon {
                transition: transform 0.2s;
                color: var(--vscode-text-muted);
            }

            .expand-icon.rotated { transform: rotate(180deg); }

            .card-body {
                padding: 0 12px 12px 12px;
                overflow: hidden;
            }

            .card-instructions {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                margin: 0;
                padding: 10px;
                background: rgba(0,0,0,0.2);
                border-radius: 4px;
                max-height: 300px;
                overflow-y: auto;
            }

            .card-output {
                margin-top: 10px;
                padding: 10px;
                background: rgba(137, 209, 133, 0.1);
                border-radius: 4px;
                border-left: 3px solid var(--vscode-success);
            }

            .card-output-label {
                font-size: 10px;
                color: var(--vscode-success);
                text-transform: uppercase;
                letter-spacing: 0.5px;
                margin-bottom: 6px;
            }

            .card-output-text {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 11px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-wrap: break-word;
                color: var(--vscode-text);
                max-height: 200px;
                overflow-y: auto;
            }

            /* Empty state */
            .empty-column {
                display: flex;
                align-items: center;
                justify-content: center;
                height: 80px;
                color: var(--vscode-text-muted);
                font-style: italic;
                font-size: 12px;
            }

            /* Modal */
            .modal-overlay {
                position: fixed;
                top: 0; left: 0; right: 0; bottom: 0;
                background: rgba(0, 0, 0, 0.6);
                z-index: 1000;
                display: flex;
                align-items: center;
                justify-content: center;
            }

            .modal {
                background: var(--vscode-sidebar);
                border: 1px solid var(--vscode-border);
                border-radius: 8px;
                max-width: 500px;
                width: 90%;
                box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
            }

            .modal-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 16px;
                border-bottom: 1px solid var(--vscode-border);
            }

            .modal-title { font-size: 14px; font-weight: 600; }

            .modal-close {
                background: transparent;
                border: none;
                color: var(--vscode-text-muted);
                cursor: pointer;
                padding: 4px;
                border-radius: 4px;
            }

            .modal-close:hover {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .modal-body { padding: 16px; }

            .modal-body textarea {
                width: 100%;
                min-height: 120px;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 12px;
                resize: vertical;
                outline: none;
            }

            .modal-body textarea:focus { border-color: var(--vscode-accent); }

            .modal-footer {
                padding: 12px 16px;
                border-top: 1px solid var(--vscode-border);
                display: flex;
                justify-content: flex-end;
                gap: 8px;
            }

            .btn {
                padding: 8px 16px;
                border-radius: 4px;
                font-size: 12px;
                cursor: pointer;
                border: 1px solid var(--vscode-border);
                transition: all 0.15s;
            }

            .btn-primary {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
                color: #fff;
            }

            .btn-primary:hover { background: var(--vscode-accent-hover); }
            .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

            .btn-secondary {
                background: var(--vscode-input-bg);
                color: var(--vscode-text);
            }

            .btn-secondary:hover { background: #4a4a4a; }

            /* Code block for instructions */
            .code-block {
                background: var(--vscode-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                overflow: hidden;
            }

            .code-header {
                display: flex;
                align-items: center;
                justify-content: space-between;
                padding: 8px 12px;
                background: var(--vscode-input-bg);
                border-bottom: 1px solid var(--vscode-border);
            }

            .code-label {
                font-size: 11px;
                color: var(--vscode-text-muted);
                text-transform: uppercase;
                letter-spacing: 0.5px;
            }

            .copy-btn {
                background: transparent;
                border: 1px solid var(--vscode-border);
                color: var(--vscode-text);
                padding: 4px 10px;
                font-size: 11px;
                cursor: pointer;
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 4px;
                transition: all 0.15s;
            }

            .copy-btn:hover {
                background: var(--vscode-accent);
                border-color: var(--vscode-accent);
            }

            .copy-btn.copied {
                background: var(--vscode-success);
                border-color: var(--vscode-success);
                color: #1e1e1e;
            }

            .code-content {
                padding: 16px;
            }

            .code-content pre {
                font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
                font-size: 13px;
                line-height: 1.6;
                color: var(--vscode-text);
                margin: 0;
                white-space: pre-wrap;
                word-wrap: break-word;
            }

            .modal.modal-wide {
                max-width: 600px;
            }

            /* Animations */
            .fade-enter-active, .fade-leave-active {
                transition: opacity 0.2s, transform 0.2s;
            }
            .fade-enter-from, .fade-leave-to {
                opacity: 0;
                transform: translateY(-10px);
            }

            [x-cloak] { display: none !important; }

            /* Toast notifications */
            .toast-container { position: fixed; top: 16px; right: 16px; z-index: 9999; display: flex; flex-direction: column; gap: 8px; }
            .toast { background: var(--vscode-sidebar); border: 1px solid var(--vscode-border); border-radius: 6px; padding: 10px 16px; font-size: 12px; color: var(--vscode-text); box-shadow: 0 4px 12px rgba(0,0,0,0.4); display: flex; align-items: center; gap: 8px; animation: slideIn 0.2s ease; min-width: 200px; }
            .toast.toast-success { border-left: 3px solid var(--vscode-success); }
            .toast.toast-error { border-left: 3px solid var(--vscode-error); }
            .toast.toast-info { border-left: 3px solid var(--vscode-info); }
            @keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
            @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

            /* Confirm dialog */
            .confirm-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.6); z-index: 2000; display: flex; align-items: center; justify-content: center; }
            .confirm-dialog { background: var(--vscode-sidebar); border: 1px solid var(--vscode-border); border-radius: 8px; padding: 20px; max-width: 400px; width: 90%; box-shadow: 0 8px 32px rgba(0,0,0,0.4); }
            .confirm-dialog p { margin-bottom: 16px; font-size: 13px; }
            .confirm-actions { display: flex; justify-content: flex-end; gap: 8px; }

            /* Filter bar */
            .filter-bar { display: flex; align-items: center; gap: 10px; padding: 8px 20px; background: var(--vscode-sidebar); border-bottom: 1px solid var(--vscode-border); flex-shrink: 0; }
            .filter-bar input { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 5px 10px; font-size: 12px; width: 220px; outline: none; }
            .filter-bar input:focus { border-color: var(--vscode-accent); }
            .filter-bar select { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 5px 8px; font-size: 12px; outline: none; cursor: pointer; }
            .filter-bar .filter-label { font-size: 11px; color: var(--vscode-text-muted); }
            .filter-bar .task-count { font-size: 11px; color: var(--vscode-text-muted); margin-left: auto; }

            /* Type/priority badges on cards */
            .card-badges { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 6px; padding: 0 12px 8px; }
            .badge { display: inline-block; padding: 1px 6px; border-radius: 3px; font-size: 10px; font-weight: 500; text-transform: uppercase; }
            .badge-type { background: rgba(55,148,255,0.15); color: var(--vscode-info); }
            .badge-priority-high { background: rgba(244,135,113,0.15); color: var(--vscode-error); }
            .badge-priority-medium { background: rgba(204,167,0,0.15); color: var(--vscode-warning); }
            .badge-priority-low { background: rgba(137,209,133,0.15); color: var(--vscode-success); }
            .badge-agent { background: rgba(204,204,204,0.1); color: var(--vscode-text-muted); }
            .badge-blocked { background: rgba(244,135,113,0.2); color: var(--vscode-error); }
            .badge-deps { background: rgba(55,148,255,0.1); color: var(--vscode-text-muted); font-size: 9px; }
            .badge-approval { background: rgba(204,167,0,0.15); color: var(--vscode-warning); }
            .badge-status-v2 { font-size: 9px; letter-spacing: 0.3px; font-weight: 600; }
            .badge-v2-QUEUED { background: rgba(55,148,255,0.15); color: #56b6f7; }
            .badge-v2-CLAIMED { background: rgba(204,167,0,0.15); color: #cca700; }
            .badge-v2-RUNNING { background: rgba(55,148,255,0.2); color: #3794ff; }
            .badge-v2-SUCCEEDED { background: rgba(137,209,133,0.15); color: #89d185; }
            .badge-v2-FAILED { background: rgba(244,135,113,0.2); color: #f48771; }
            .badge-v2-NEEDS_REVIEW { background: rgba(204,167,0,0.2); color: #e8c600; }
            .badge-v2-CANCELLED { background: rgba(133,133,133,0.15); color: #999; }

            /* Task detail drawer */
            .detail-drawer { position: fixed; top: 0; right: 0; bottom: 0; width: 480px; background: var(--vscode-sidebar); border-left: 1px solid var(--vscode-border); z-index: 900; box-shadow: -4px 0 16px rgba(0,0,0,0.3); display: flex; flex-direction: column; transform: translateX(100%); transition: transform 0.2s ease; }
            .detail-drawer.open { transform: translateX(0); }
            .drawer-header { display: flex; align-items: center; justify-content: space-between; padding: 16px; border-bottom: 1px solid var(--vscode-border); flex-shrink: 0; }
            .drawer-header h2 { font-size: 14px; font-weight: 600; }
            .drawer-body { flex: 1; overflow-y: auto; padding: 16px; }
            .drawer-section { margin-bottom: 16px; }
            .drawer-section h3 { font-size: 12px; color: var(--vscode-text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
            .drawer-section pre { font-family: 'Cascadia Code','Fira Code',Consolas,monospace; font-size: 11px; line-height: 1.5; white-space: pre-wrap; word-wrap: break-word; color: var(--vscode-text); padding: 10px; background: rgba(0,0,0,0.2); border-radius: 4px; margin: 0; }
            .drawer-meta { display: grid; grid-template-columns: 100px 1fr; gap: 6px; font-size: 12px; }
            .drawer-meta dt { color: var(--vscode-text-muted); }
            .drawer-meta dd { color: var(--vscode-text); margin: 0; }
            .drawer-actions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 12px; }
            .drawer-actions button { background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); color: var(--vscode-text); padding: 4px 10px; border-radius: 4px; font-size: 11px; cursor: pointer; }
            .drawer-actions button:hover { border-color: var(--vscode-accent); }
            .drawer-actions button.danger { color: var(--vscode-error); }
            .drawer-actions button.danger:hover { border-color: var(--vscode-error); }

            /* Task hierarchy - indentation for child tasks */
            .task-card[data-depth="1"] { margin-left: 20px; border-left: 3px solid var(--vscode-accent); }
            .task-card[data-depth="2"] { margin-left: 40px; border-left: 3px solid var(--vscode-info); }
            .task-card[data-depth="3"] { margin-left: 60px; border-left: 3px solid var(--vscode-warning); }

            .parent-badge {
                display: inline-flex;
                align-items: center;
                gap: 4px;
                font-size: 10px;
                color: var(--vscode-text-muted);
                background: var(--vscode-input-bg);
                padding: 2px 6px;
                border-radius: 4px;
            }

            .parent-badge svg {
                width: 10px;
                height: 10px;
            }

            /* Parent task select in modal */
            .modal-body select {
                width: 100%;
                background: var(--vscode-input-bg);
                border: 1px solid var(--vscode-border);
                border-radius: 6px;
                color: var(--vscode-text);
                font-family: inherit;
                font-size: 13px;
                padding: 10px 12px;
                outline: none;
                cursor: pointer;
            }

            .modal-body select:focus { border-color: var(--vscode-accent); }

            .modal-body label {
                display: block;
                font-size: 12px;
                color: var(--vscode-text-muted);
                margin-bottom: 6px;
            }

            .form-group {
                margin-bottom: 12px;
            }

            /* Agent Status Styles */
            .status-badge {
                border-radius: 10px;
                font-size: 10px;
                font-weight: 500;
                padding: 2px 6px;
                text-transform: uppercase;
            }

            .status-badge.status-online {
                background: var(--vscode-charts-green);
                color: white;
            }

            .status-badge.status-offline {
                background: var(--vscode-charts-red);
                color: white;
            }

            .status-badge.status-busy {
                background: var(--vscode-charts-orange);
                color: white;
            }

            .status-badge.status-idle {
                background: var(--vscode-charts-blue);
                color: white;
            }

            .agent-item:last-child {
                border-bottom: none;
            }

            .agent-status [x-cloak] { display: none !important; }

            /* Footer */
            .footer {
                background: var(--vscode-sidebar);
                border-top: 1px solid var(--vscode-border);
                padding: 10px 20px;
                text-align: center;
                font-size: 12px;
                color: var(--vscode-text-muted);
                flex-shrink: 0;
            }

            .footer a {
                color: var(--vscode-accent);
                text-decoration: none;
            }

            .footer a:hover {
                text-decoration: underline;
            }
        </style>
    </head>
    <body x-data="kanbanApp()" x-init="init()">
        <!-- Toast notifications -->
        <div class="toast-container">
            <template x-for="(t, i) in toasts" :key="i">
                <div class="toast" :class="'toast-' + t.type" x-text="t.message"
                     x-init="setTimeout(() => toasts.splice(toasts.indexOf(t), 1), 3000)"></div>
            </template>
        </div>

        <!-- Confirm dialog -->
        <div class="confirm-overlay" x-show="confirmDialog.show" x-cloak @click.self="confirmDialog.show = false">
            <div class="confirm-dialog">
                <p x-text="confirmDialog.message"></p>
                <div class="confirm-actions">
                    <button class="btn btn-secondary" @click="confirmDialog.show = false">Cancel</button>
                    <button class="btn btn-primary" @click="confirmDialog.onConfirm(); confirmDialog.show = false">Confirm</button>
                </div>
            </div>
        </div>

        <!-- Task detail drawer -->
        <div class="detail-drawer" :class="{ 'open': selectedTask }" x-show="selectedTask" x-cloak>
            <div class="drawer-header">
                <h2 x-text="selectedTask ? '#' + selectedTask.id + ' Details' : ''"></h2>
                <button class="modal-close" @click="selectedTask = null">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                        <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                    </svg>
                </button>
            </div>
            <div class="drawer-body" x-show="selectedTask">
                <div class="drawer-section">
                    <h3>Metadata</h3>
                    <dl class="drawer-meta">
                        <dt>ID</dt><dd x-text="selectedTask ? selectedTask.id : ''"></dd>
                        <dt>Title</dt><dd x-text="selectedTask ? (selectedTask.title || '—') : ''"></dd>
                        <dt>Status</dt><dd x-html="selectedTask ? stsBadge(selectedTask.status_v2 || selectedTask.status) : ''"></dd>
                        <dt>Created</dt><dd><span x-text="selectedTask ? fmtAgo(selectedTask.created_at) : ''" :title="selectedTask ? selectedTask.created_at : ''"></span></dd>
                        <dt>Updated</dt><dd><span x-text="selectedTask ? fmtAgo(selectedTask.updated_at) : '—'" :title="selectedTask ? (selectedTask.updated_at || '') : ''"></span></dd>
                        <dt>Parent</dt><dd x-html="selectedTask && selectedTask.parent_task_id ? entityLink('/board?task='+selectedTask.parent_task_id, '#'+selectedTask.parent_task_id) : '—'"></dd>
                        <dt>Priority</dt><dd x-text="selectedTask ? (selectedTask.priority || 50) : ''"></dd>
                        <dt>Type</dt><dd x-text="selectedTask ? (selectedTask.type || '—') : ''"></dd>
                        <dt>Agent</dt><dd x-html="selectedTask && selectedTask.agent_id ? entityLink('/agents?id='+encodeURIComponent(selectedTask.agent_id), selectedTask.agent_id.substring(0,20)) : '—'"></dd>
                        <dt>Tags</dt><dd x-text="selectedTask && selectedTask.tags ? selectedTask.tags.join(', ') : '—'"></dd>
                        <dt>Blocked</dt><dd x-html="selectedTask && selectedTask.blocked ? stsBadge('BLOCKED') : '—'"></dd>
                        <dt>Dependencies</dt><dd x-html="selectedTask && selectedTask.dependency_count ? entityLink('/dependencies?task='+selectedTask.id, selectedTask.dependency_count+' dep'+(selectedTask.dependency_count>1?'s':'')) : '—'"></dd>
                    </dl>
                </div>
                <div class="drawer-section">
                    <h3>Quick Actions</h3>
                    <div class="drawer-actions">
                        <button @click="claimSelectedTask" x-show="selectedTask && selectedTask.status === 'NOT_PICKED'">Claim</button>
                        <button @click="completeSelectedTask" x-show="selectedTask && selectedTask.status === 'IN_PROGRESS'">Complete</button>
                        <button class="danger" @click="failSelectedTask" x-show="selectedTask && selectedTask.status === 'IN_PROGRESS'">Fail</button>
                        <button @click="duplicateSelectedTask">Duplicate</button>
                        <button class="danger" @click="deleteTask(selectedTask.id); selectedTask = null">Delete</button>
                    </div>
                </div>
                <div class="drawer-section">
                    <h3>Related</h3>
                    <div style="font-size:12px;display:flex;flex-direction:column;gap:6px;">
                        <template x-if="selectedTask && tasks.filter(t => t.parent_task_id === selectedTask.id).length > 0">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Child Tasks</div>
                                <template x-for="child in tasks.filter(t => t.parent_task_id === selectedTask.id)" :key="child.id">
                                    <div style="display:flex;align-items:center;gap:6px;padding:2px 0;">
                                        <a :href="'/board?task='+child.id" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" @click.prevent="selectedTask = child" x-text="'#'+child.id"></a>
                                        <span x-text="child.title || child.instructions?.substring(0,30)+'...' || ''" style="color:var(--vscode-text-muted);"></span>
                                        <span x-html="stsBadge(child.status_v2 || child.status)" style="margin-left:auto;"></span>
                                    </div>
                                </template>
                            </div>
                        </template>
                        <template x-if="selectedTask && selectedTask.parent_task_id">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Parent</div>
                                <div style="display:flex;align-items:center;gap:6px;">
                                    <a :href="'/board?task='+selectedTask.parent_task_id" style="color:var(--vscode-accent);text-decoration:none;" @click.prevent="selectedTask = tasks.find(t => t.id === selectedTask.parent_task_id) || selectedTask" x-text="'#'+selectedTask.parent_task_id"></a>
                                    <span x-text="(tasks.find(t => t.id === selectedTask.parent_task_id) || {}).title || ''" style="color:var(--vscode-text-muted);"></span>
                                </div>
                            </div>
                        </template>
                        <template x-if="selectedTask && selectedTask.agent_id">
                            <div>
                                <div style="color:var(--vscode-text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.03em;margin-bottom:4px;">Assigned Agent</div>
                                <a :href="'/agents?id='+encodeURIComponent(selectedTask.agent_id)" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" x-text="selectedTask.agent_id.substring(0,24)"></a>
                            </div>
                        </template>
                        <template x-if="selectedTask && !tasks.filter(t => t.parent_task_id === selectedTask?.id).length && !selectedTask.parent_task_id && !selectedTask.agent_id">
                            <div style="color:var(--vscode-text-muted);font-style:italic;">No related entities</div>
                        </template>
                    </div>
                </div>
                <div class="drawer-section">
                    <h3>Instructions</h3>
                    <pre x-text="selectedTask ? selectedTask.instructions : ''"></pre>
                </div>
                <div class="drawer-section" x-show="selectedTask && selectedTask.output">
                    <h3>Output</h3>
                    <pre x-text="selectedTask ? (selectedTask.output || '') : ''"></pre>
                </div>
            </div>
        </div>

        <!-- Create Task Modal -->
        <div class="modal-overlay"
             x-show="showModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showModal = false"
             @keydown.escape.window="showModal = false">
            <div class="modal" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Create New Task</span>
                    <button class="modal-close" @click="showModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <form @submit.prevent="createTask">
                    <div class="modal-body">
                        <div class="form-group">
                            <label>Title (optional)</label>
                            <input type="text" x-model="newTaskTitle" placeholder="Brief task title"
                                   style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-family:inherit;font-size:13px;padding:10px 12px;outline:none;">
                        </div>
                        <div class="form-group">
                            <label>Parent Task (optional)</label>
                            <select x-model="newTaskParentId">
                                <option value="">No parent (standalone task)</option>
                                <template x-for="task in tasks" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + task.instructions.substring(0, 50) + (task.instructions.length > 50 ? '...' : '')"></option>
                                </template>
                            </select>
                        </div>
                        <div style="display:flex;gap:10px;margin-bottom:12px;">
                            <div class="form-group" style="flex:1;">
                                <label>Type (optional)</label>
                                <select x-model="newTaskType" style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                    <option value="">None</option>
                                    <option value="analyze">Analyze</option>
                                    <option value="modify">Modify</option>
                                    <option value="test">Test</option>
                                    <option value="review">Review</option>
                                    <option value="doc">Doc</option>
                                    <option value="plan">Plan</option>
                                    <option value="feature">Feature</option>
                                    <option value="bug">Bug</option>
                                    <option value="chore">Chore</option>
                                    <option value="research">Research</option>
                                    <option value="refactor">Refactor</option>
                                </select>
                            </div>
                            <div class="form-group" style="flex:1;">
                                <label>Priority (optional)</label>
                                <select x-model="newTaskPriority" style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                    <option value="">Normal</option>
                                    <option value="high">High</option>
                                    <option value="medium">Medium</option>
                                    <option value="low">Low</option>
                                </select>
                            </div>
                        </div>
                        <div class="form-group">
                            <label>Tags (comma-separated, optional)</label>
                            <input type="text" x-model="newTaskTags" placeholder="e.g. backend, urgent, v2"
                                   style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-family:inherit;font-size:13px;padding:10px 12px;outline:none;">
                        </div>
                        <div class="form-group">
                            <label>Dependencies (tasks this depends on)</label>
                            <select x-model="newTaskDependencies" multiple
                                    style="width:100%;min-height:60px;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:6px 8px;outline:none;">
                                <template x-for="task in tasks.filter(t => t.status !== 'complete')" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + (task.title || task.instructions.substring(0, 50))"></option>
                                </template>
                            </select>
                            <small style="color:var(--vscode-descriptionForeground);font-size:11px;">Hold Ctrl/Cmd to select multiple</small>
                        </div>
                        <div class="form-group">
                            <label>Template (optional)</label>
                            <select x-model="newTaskTemplateId" @change="applyTemplate()"
                                    style="width:100%;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;color:var(--vscode-text);font-size:13px;padding:10px 12px;outline:none;cursor:pointer;">
                                <option value="">No template</option>
                                <template x-for="tmpl in templates" :key="tmpl.id">
                                    <option :value="tmpl.id" x-text="tmpl.name + (tmpl.description ? ' — ' + tmpl.description : '')"></option>
                                </template>
                            </select>
                        </div>
                        <div class="form-group" style="display:flex;align-items:center;gap:8px;">
                            <input type="checkbox" id="requiresApproval" x-model="newTaskRequiresApproval"
                                   style="width:16px;height:16px;cursor:pointer;">
                            <label for="requiresApproval" style="margin:0;cursor:pointer;">Requires approval before completion</label>
                        </div>
                        <div class="form-group">
                            <label>Instructions</label>
                            <textarea
                                x-model="newTaskInstructions"
                                placeholder="Enter task instructions..."
                                required
                                x-ref="taskInput"></textarea>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" @click="showModal = false">Cancel</button>
                        <button type="submit" class="btn btn-primary" :disabled="!newTaskInstructions.trim()">Create Task</button>
                    </div>
                </form>
            </div>
        </div>

        <!-- Instructions Modal -->
        <div class="modal-overlay"
             x-show="showInstructionsModal"
             x-cloak
             x-transition:enter="fade-enter-active"
             x-transition:leave="fade-leave-active"
             @click.self="showInstructionsModal = false"
             @keydown.escape.window="showInstructionsModal = false">
            <div class="modal modal-wide" @click.stop>
                <div class="modal-header">
                    <span class="modal-title">Initial Copilot Instructions</span>
                    <button class="modal-close" @click="showInstructionsModal = false">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path fill-rule="evenodd" clip-rule="evenodd" d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.707.708L7.293 8l-3.646 3.646.707.708L8 8.707z"/>
                        </svg>
                    </button>
                </div>
                <div class="modal-body">
                    <div class="code-block">
                        <div class="code-header">
                            <span class="code-label">Paste this to Copilot</span>
                            <button class="copy-btn" :class="{ 'copied': copiedInstructions }" @click="copyInstructions">
                                <svg x-show="!copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M4 4l1-1h5.414L14 6.586V14l-1 1H5l-1-1V4zm9 3l-3-3H5v10h8V7zM3 1L2 2v10l1 1V2h6.414l-1-1H3z"/>
                                </svg>
                                <svg x-show="copiedInstructions" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                    <path fill-rule="evenodd" clip-rule="evenodd" d="M14.431 3.323l-8.47 10-.79-.036-3.35-4.77.818-.574 2.978 4.24 8.051-9.506.763.646z"/>
                                </svg>
                                <span x-text="copiedInstructions ? 'Copied!' : 'Copy'"></span>
                            </button>
                        </div>
                        <div class="code-content">
                            <pre x-ref="instructionsText">
                            See http://127.0.0.1:8080/instructions use curl to get the instructions and proceed with it. Do not stop when there are no new tasks!
                            </pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="header">
            <svg class="header-icon" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8.5 1a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 1a5.5 5.5 0 1 0 0 11 5.5 5.5 0 0 0 0-11zm-2 3.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm0 2a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5z"/>
            </svg>
            <span class="header-title">Cocopilot</span>
            <nav class="header-nav" aria-label="Primary">
                <a href="/dashboard">Dashboard</a>
                <a href="/board" class="active">Work Queue</a>
                <a href="/runs">Runs</a>
                <a href="/agents">Agents</a>
                <a href="/repo">Repo</a>
                <a href="/events-browser">Events</a>
                <a href="/settings">Settings</a>
                <div class="nav-more" x-data="{ open: false }" @click.outside="open = false">
                    <button class="nav-more-btn" @click="open = !open">More &#9662;</button>
                    <div class="nav-more-dropdown" x-show="open" x-cloak @click="open = false">
                        <a href="/graphs/tasks">Task Graph</a>
                        <a href="/dependencies">Dependencies</a>
                        <a href="/memory">Memory</a>
                        <a href="/policies">Policies</a>
                        <a href="/context-packs">Context Packs</a>
                        <a href="/audit">Audit Log</a>
                        <a href="/health">Health</a>
                    </div>
                </div>
            </nav>
            <div class="header-actions">
                <div class="workdir-input">
                    <label for="project">Project:</label>
                    <select id="project" x-model="currentProject" @change="onProjectChange" style="background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; color: var(--vscode-text); padding: 6px 10px; font-size: 12px; min-width: 150px; cursor: pointer;">
                        <template x-for="project in projects" :key="project.id">
                            <option :value="project.id" x-text="project.name"></option>
                        </template>
                    </select>
                </div>
                <div class="workdir-input">
                    <label>Agents:</label>
                    <div class="agent-status" x-data="{ showAgentDetails: false }" @click="showAgentDetails = !showAgentDetails" style="position: relative; cursor: pointer;">
                        <div style="display: flex; align-items: center; gap: 6px; background: var(--vscode-input-bg); border: 1px solid var(--vscode-border); border-radius: 4px; padding: 6px 10px; font-size: 12px; min-width: 120px;">
                            <span x-text="agents.filter(a => a.status === 'ONLINE').length"></span>
                            <span style="color: var(--vscode-text-muted);">online</span>
                            <span style="color: var(--vscode-text-muted); font-size: 10px;">▼</span>
                        </div>
                        <div x-show="showAgentDetails" x-cloak style="position: absolute; top: 100%; right: 0; background: var(--vscode-dropdown-bg); border: 1px solid var(--vscode-border); border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); z-index: 1000; min-width: 200px; max-height: 300px; overflow-y: auto;">
                            <template x-for="agent in agents" :key="agent.id">
                                <div class="agent-item" style="padding: 8px 12px; border-bottom: 1px solid var(--vscode-border); display: flex; justify-content: space-between; align-items: center;">
                                    <span x-text="agent.name" style="font-size: 12px; font-weight: 500;"></span>
                                    <span :class="'status-badge status-' + agent.status.toLowerCase()" x-text="agent.status" style="font-size: 10px; padding: 2px 6px; border-radius: 10px; font-weight: 500;"></span>
                                </div>
                            </template>
                            <div style="padding: 12px; text-align: center; color: var(--vscode-text-muted); font-size: 12px;">
                                No agents connected
                            </div>
                        </div>
                    </div>
                </div>
                <div class="workdir-input">
                    <label for="workdir">Workdir:</label>
                    <input type="text" id="workdir" x-model="workdir" @change="updateWorkdir" placeholder="/path/to/workdir">
                </div>
                <button class="header-btn" @click="showInstructionsModal = true">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14.5 2H9l-.35.15-.65.64-.65-.64L7 2H1.5l-.5.5v10l.5.5h5.29l.86.85h1.71l.85-.85h5.29l.5-.5v-10l-.5-.5zm-7 10.32l-.18-.17L7 12H2V3h4.79l.74.74-.03 8.58zM14 12H9l-.35.15-.14.13V3.7l.7-.7H14v9z"/>
                    </svg>
                    Initial Instructions
                </button>
                <button class="header-btn" @click="openModal">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14 7v1H8v6H7V8H1V7h6V1h1v6h6z"/>
                    </svg>
                    New Task
                </button>
            </div>
        </div>

        <!-- Filter bar -->
        <div class="filter-bar">
            <span class="filter-label">Filter:</span>
            <input type="text" placeholder="Search tasks... (press /)" x-model="searchQuery">
            <select x-model="filterType">
                <option value="">All types</option>
                <option value="ANALYZE">Analyze</option>
                <option value="MODIFY">Modify</option>
                <option value="TEST">Test</option>
                <option value="REVIEW">Review</option>
                <option value="DOC">Doc</option>
                <option value="PLAN">Plan</option>
                <option value="RELEASE">Release</option>
                <option value="FEATURE">Feature</option>
                <option value="BUG">Bug</option>
                <option value="CHORE">Chore</option>
                <option value="REFACTOR">Refactor</option>
                <option value="RESEARCH">Research</option>
            </select>
            <select x-model="filterPriority">
                <option value="">All priorities</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
            </select>
            <select x-model="filterAgent">
                <option value="">All agents</option>
                <template x-for="a in [...new Set(tasks.map(t => t.agent_id).filter(Boolean))]" :key="a">
                    <option :value="a" x-text="a"></option>
                </template>
            </select>
            <span class="filter-label">Sort:</span>
            <select x-model="sortBy">
                <option value="newest">Newest first</option>
                <option value="oldest">Oldest first</option>
                <option value="priority">Priority</option>
                <option value="id">By ID</option>
            </select>
            <label style="display:flex;align-items:center;gap:4px;font-size:11px;cursor:pointer;color:var(--vscode-text-muted);">
                <input type="checkbox" x-model="filterBlocked" style="margin:0;"> Blocked only
            </label>
            <select x-model="filterStatusV2" style="font-size:11px;padding:2px 6px;background:var(--vscode-input-bg);border:1px solid var(--vscode-border);color:var(--vscode-text);border-radius:4px;">
                <option value="">All states</option>
                <option value="QUEUED">Queued</option>
                <option value="CLAIMED">Claimed</option>
                <option value="RUNNING">Running</option>
                <option value="SUCCEEDED">Succeeded</option>
                <option value="FAILED">Failed</option>
                <option value="NEEDS_REVIEW">Needs Review</option>
                <option value="CANCELLED">Cancelled</option>
            </select>
            <span class="task-count" x-text="tasks.length + ' task' + (tasks.length !== 1 ? 's' : '') + (searchQuery || filterType || filterPriority || filterAgent || filterBlocked || filterStatusV2 ? ' (' + filteredCount + ' matching)' : '')"></span>
            <span x-show="sseReconnecting" x-cloak style="color:var(--vscode-warning);font-size:11px;display:flex;align-items:center;gap:4px;" title="Reconnecting to server...">
                <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor" style="animation:spin 1s linear infinite;"><path d="M8 1a7 7 0 1 0 7 7h-1.5A5.5 5.5 0 1 1 8 2.5V1z"/></svg>
                Reconnecting
            </span>
            <span x-show="sseConnected && !sseReconnecting" style="color:var(--vscode-success);font-size:10px;" title="Connected">&#9679;</span>
        </div>

        <!-- Dashboard stats bar -->
        <div style="display:flex;gap:12px;padding:10px 20px;background:var(--vscode-sidebar);border-bottom:1px solid var(--vscode-border);flex-shrink:0;">
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-warning);" x-text="todoTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Pending</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-info);" x-text="progressTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">In Progress</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-success);" x-text="doneTasks.length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Done</div>
            </div>
            <div style="background:var(--vscode-input-bg);border:1px solid var(--vscode-border);border-radius:6px;padding:8px 14px;flex:1;text-align:center;">
                <div style="font-size:20px;font-weight:700;color:var(--vscode-accent);" x-text="agents.filter(a => a.status === 'ONLINE').length"></div>
                <div style="font-size:10px;color:var(--vscode-text-muted);text-transform:uppercase;">Agents Online</div>
            </div>
        </div>

        <div class="kanban-board">
            <!-- To Do Column -->
            <div class="kanban-column col-todo"
                 :class="{ 'drag-over': dragOverColumn === 'NOT_PICKED' }"
                 @dragover.prevent="dragOverColumn = 'NOT_PICKED'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('NOT_PICKED')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>
                        <path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0z"/>
                    </svg>
                    <span class="column-title">To Do</span>
                    <span class="column-count" x-text="todoTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in todoTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-blocked" x-show="task.blocked">BLOCKED</span>
                                <span class="badge badge-deps" x-show="task.dependency_count" x-text="task.dependency_count + ' dep' + (task.dependency_count > 1 ? 's' : '')"></span>
                                <span class="badge badge-approval" x-show="task.requires_approval">APPROVAL</span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="todoTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);margin-bottom:8px;">Queue is empty</p>
                        <button class="btn btn-primary" @click="openModal" style="font-size:11px;padding:4px 12px;">+ Create Task</button>
                    </div>
                </div>
            </div>

            <!-- In Progress Column -->
            <div class="kanban-column col-progress"
                 :class="{ 'drag-over': dragOverColumn === 'IN_PROGRESS' }"
                 @dragover.prevent="dragOverColumn = 'IN_PROGRESS'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('IN_PROGRESS')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M8 0a8 8 0 1 0 8 8A8 8 0 0 0 8 0zm0 14a6 6 0 1 1 6-6 6 6 0 0 1-6 6z" opacity=".3"/>
                        <path d="M8 0v2a6 6 0 0 1 6 6h2a8 8 0 0 0-8-8z"/>
                    </svg>
                    <span class="column-title">In Progress</span>
                    <span class="column-count" x-text="progressTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in progressTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-blocked" x-show="task.blocked">BLOCKED</span>
                                <span class="badge badge-deps" x-show="task.dependency_count" x-text="task.dependency_count + ' dep' + (task.dependency_count > 1 ? 's' : '')"></span>
                                <span class="badge badge-approval" x-show="task.requires_approval">APPROVAL</span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="progressTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);">No tasks being worked on</p>
                    </div>
                </div>
            </div>

            <!-- Done Column -->
            <div class="kanban-column col-done"
                 :class="{ 'drag-over': dragOverColumn === 'COMPLETE' }"
                 @dragover.prevent="dragOverColumn = 'COMPLETE'"
                 @dragleave="dragOverColumn = null"
                 @drop="dropTask('COMPLETE')">
                <div class="column-header">
                    <svg class="column-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
                    </svg>
                    <span class="column-title">Done</span>
                    <span class="column-count" x-text="doneTasks.length"></span>
                </div>
                <div class="column-body">
                    <template x-for="task in doneTasks" :key="task.id">
                        <div class="task-card"
                             x-show="matchesSearch(task)"
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id, 'task-blocked': task.blocked }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag"
                             @dblclick="selectedTask = task">
                            <div class="card-header" @click="selectedTask = task">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-title" x-show="task.title" x-text="(task.title||'').substring(0, 50) + ((task.title||'').length > 50 ? '...' : '')" :title="task.title" style="font-weight:600;color:#ccc;font-size:12px;"></span>
                                    <span class="card-preview" x-text="(task.instructions||'').substring(0, 40) + ((task.instructions||'').length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id && !task.title"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="formatTime(task.updated_at || task.created_at)" :title="task.updated_at || task.created_at"></span>
                                    <button class="delete-btn" @click.stop="deleteTask(task.id)" title="Delete task">
                                        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                                            <path fill-rule="evenodd" d="M5.75 3V1.5h4.5V3h3.75v1.5H2V3h3.75zm-.5 5.25a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zm4 0a.75.75 0 011.5 0v4.5a.75.75 0 01-1.5 0v-4.5zM3.5 5v9.5h9V5h-9z"/>
                                        </svg>
                                    </button>
                                    <svg class="expand-icon" :class="{ 'rotated': expandedTasks.includes(task.id) }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                                        <path fill-rule="evenodd" d="M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z"/>
                                    </svg>
                                </div>
                            </div>
                            <div class="card-badges">
                                <span class="badge badge-type" x-show="task.type" x-text="task.type"></span>
                                <span class="badge" :class="task.priority >= 70 ? 'badge-priority-high' : task.priority >= 40 ? 'badge-priority-medium' : 'badge-priority-low'" x-show="task.priority" x-text="task.priority >= 70 ? 'HIGH' : task.priority >= 40 ? 'MED' : 'LOW'"></span>
                                <span class="badge badge-agent" x-show="task.agent_id" x-text="task.agent_id"></span>
                                <span class="badge badge-status-v2" :class="'badge-v2-' + (task.status_v2 || '')" x-show="task.status_v2 && task.status_v2 !== task.status" x-text="task.status_v2"></span>
                            </div>
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                                <div class="card-output" x-show="task.output">
                                    <div class="card-output-label">Output</div>
                                    <pre class="card-output-text" x-text="task.output"></pre>
                                </div>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="doneTasks.length === 0">
                        <p style="color:var(--vscode-text-muted);">Nothing completed yet</p>
                    </div>
                </div>
            </div>
        </div>

        <footer class="footer">
            Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a>
        </footer>

        <script>
            // Shared helpers for entity rendering
            function fmtAgo(iso){if(!iso)return '—';const s=Math.floor((Date.now()-new Date(iso))/1000);if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';return new Date(iso).toLocaleDateString();}
            function stsBadge(s){const m={SUCCEEDED:'ok',COMPLETED:'ok',DONE:'ok',FAILED:'error',CANCELLED:'warn',QUEUED:'info',CLAIMED:'warn',RUNNING:'warn',IN_PROGRESS:'warn',TODO:'info',PENDING:'info',NEEDS_REVIEW:'warn'};const c=m[s]||'muted';return '<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.03em;background:'+(c==='ok'?'#1a3a1a':c==='error'?'#3a1919':c==='warn'?'#3a3519':c==='info'?'#19293a':'var(--vscode-input-bg)')+';color:var(--vscode-'+(c==='ok'?'success':c==='error'?'error':c==='warn'?'warning':c==='info'?'info':'text-muted')+');border:1px solid '+(c==='ok'?'#2d5a2d':c==='error'?'#5a2525':c==='warn'?'#5a5325':c==='info'?'#254a6e':'var(--vscode-border)')+';">'+((s||'').replace(/[<>&"']/g,''))+'</span>';}
            function entityLink(href,label){return '<a href="'+href+'" style="color:var(--vscode-accent);text-decoration:none;font-size:12px;" onmouseover="this.style.textDecoration=\'underline\'" onmouseout="this.style.textDecoration=\'none\'">'+((label||'').replace(/[<>&"']/g,''))+'</a>';}
            function kanbanApp() {
                return {
                    tasks: [],
                    expandedTasks: [],
                    showModal: false,
                    showInstructionsModal: false,
                    copiedInstructions: false,
                    newTaskInstructions: '',
                    newTaskParentId: '',
                    newTaskTitle: '',
                    newTaskType: '',
                    newTaskPriority: '',
                    newTaskTags: '',
                    newTaskRequiresApproval: false,
                    newTaskDependencies: [],
                    newTaskTemplateId: '',
                    templates: [],
                    draggingId: null,
                    draggingTask: null,
                    dragOverColumn: null,
                    eventSource: null,
                    workdir: '/tmp',
                    projects: [],
                    currentProject: 'proj_default',
                    agents: [],
                    toasts: [],
                    confirmDialog: { show: false, message: '', onConfirm: () => {} },
                    searchQuery: '',
                    sortBy: 'newest',
                    selectedTask: null,
                    filterType: '',
                    filterPriority: '',
                    filterAgent: '',
                    filterStatus: '',
                    filterBlocked: false,
                    filterStatusV2: '',
                    sseConnected: false,
                    sseReconnecting: false,

                    // Toast helper
                    showToast(message, type = 'info') {
                        this.toasts.push({ message, type });
                    },

                    // Confirm dialog helper
                    showConfirm(message, onConfirm) {
                        this.confirmDialog = { show: true, message, onConfirm };
                    },

                    // Search + filter
                    matchesSearch(task) {
                        if (this.filterType && (task.type || '').toUpperCase() !== this.filterType.toUpperCase()) return false;
                        if (this.filterAgent && (task.agent_id || '') !== this.filterAgent) return false;
                        if (this.filterBlocked && !task.blocked) return false;
                        if (this.filterStatusV2 && (task.status_v2 || '') !== this.filterStatusV2) return false;
                        if (this.filterPriority) {
                            const p = task.priority || 50;
                            if (this.filterPriority === 'high' && p < 70) return false;
                            if (this.filterPriority === 'medium' && (p < 40 || p >= 70)) return false;
                            if (this.filterPriority === 'low' && p >= 40) return false;
                        }
                        if (!this.searchQuery) return true;
                        const q = this.searchQuery.toLowerCase();
                        return (
                            String(task.id).includes(q) ||
                            (task.instructions || '').toLowerCase().includes(q) ||
                            (task.type || '').toLowerCase().includes(q) ||
                            (task.agent_id || '').toLowerCase().includes(q) ||
                            (task.title || '').toLowerCase().includes(q)
                        );
                    },

                    get filteredCount() {
                        return this.tasks.filter(t => this.matchesSearch(t)).length;
                    },

                    // Compute task depth based on parent chain
                    getTaskDepth(task) {
                        let depth = 0;
                        let currentParentId = task.parent_task_id;
                        while (currentParentId && depth < 3) {
                            const parent = this.tasks.find(t => t.id === currentParentId);
                            if (!parent) break;
                            depth++;
                            currentParentId = parent.parent_task_id;
                        }
                        return depth;
                    },

                    // Organize tasks hierarchically within each status column
                    getOrganizedTasks(status) {
                        const statusTasks = this.tasks.filter(t => t.status === status);
                        const rootTasks = statusTasks.filter(t => !t.parent_task_id);
                        const childrenMap = {};

                        statusTasks.forEach(t => {
                            if (t.parent_task_id) {
                                if (!childrenMap[t.parent_task_id]) childrenMap[t.parent_task_id] = [];
                                childrenMap[t.parent_task_id].push(t);
                            }
                        });

                        const result = [];
                        const addWithChildren = (task, depth = 0) => {
                            result.push({ ...task, depth });
                            (childrenMap[task.id] || []).forEach(child => addWithChildren(child, depth + 1));
                        };

                        rootTasks.forEach(t => addWithChildren(t));

                        // Also add orphaned children (whose parent is in a different status)
                        statusTasks.forEach(t => {
                            if (t.parent_task_id && !result.find(r => r.id === t.id)) {
                                result.push({ ...t, depth: this.getTaskDepth(t) });
                            }
                        });

                        return result;
                    },

                    get todoTasks() {
                        return this.getOrganizedTasks('NOT_PICKED');
                    },

                    get progressTasks() {
                        return this.getOrganizedTasks('IN_PROGRESS');
                    },

                    get doneTasks() {
                        return this.getOrganizedTasks('COMPLETE');
                    },

                    init() {
                        // Restore persisted state from localStorage
                        try {
                            const saved = JSON.parse(localStorage.getItem('cocopilot_board_state') || '{}');
                            if (saved.searchQuery) this.searchQuery = saved.searchQuery;
                            if (saved.sortBy) this.sortBy = saved.sortBy;
                            if (saved.filterType) this.filterType = saved.filterType;
                            if (saved.filterPriority) this.filterPriority = saved.filterPriority;
                            if (saved.filterAgent) this.filterAgent = saved.filterAgent;
                            if (saved.expandedTasks) this.expandedTasks = saved.expandedTasks;
                        } catch(e) {}

                        this.connectSSE();
                        this.fetchWorkdir();
                        this.fetchProjects();
                        this.fetchAgents();
                        this.fetchTemplates();
                        
                        // Refresh agents status every 30 seconds
                        setInterval(() => {
                            this.fetchAgents();
                        }, 30000);

                        // Persist state on changes
                        this.$watch('searchQuery', () => this.persistState());
                        this.$watch('sortBy', () => this.persistState());
                        this.$watch('filterType', () => this.persistState());
                        this.$watch('filterPriority', () => this.persistState());
                        this.$watch('filterAgent', () => this.persistState());
                        this.$watch('expandedTasks', () => this.persistState());

                        // Keyboard shortcuts
                        document.addEventListener('keydown', (e) => {
                            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
                            if (e.key === 'n' && !e.ctrlKey && !e.metaKey) { e.preventDefault(); this.openModal(); }
                            if (e.key === '/' && !e.ctrlKey && !e.metaKey) { e.preventDefault(); document.querySelector('.filter-bar input')?.focus(); }
                            if (e.key === 'Escape' && this.selectedTask) { this.selectedTask = null; }
                        });
                    },

                    persistState() {
                        try {
                            localStorage.setItem('cocopilot_board_state', JSON.stringify({
                                searchQuery: this.searchQuery,
                                sortBy: this.sortBy,
                                filterType: this.filterType,
                                filterPriority: this.filterPriority,
                                filterAgent: this.filterAgent,
                                expandedTasks: this.expandedTasks,
                            }));
                        } catch(e) {}
                    },

                    async fetchProjects() {
                        try {
                            const response = await fetch('/api/v2/projects');
                            const data = await response.json();
                            this.projects = data.projects || [];
                            if (this.projects.length > 0 && this.projects.find(p => p.id === 'proj_default')) {
                                this.currentProject = 'proj_default';
                            } else if (this.projects.length > 0) {
                                this.currentProject = this.projects[0].id;
                            }
                        } catch (error) {
                            console.error('Failed to fetch projects:', error);
                        }
                    },

                    async fetchAgents() {
                        try {
                            const response = await fetch('/api/v2/agents');
                            const data = await response.json();
                            this.agents = data.agents || [];
                        } catch (error) {
                            console.error('Failed to fetch agents:', error);
                            this.agents = [];
                        }
                    },

                    async fetchTemplates() {
                        try {
                            const response = await fetch('/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/templates');
                            const data = await response.json();
                            this.templates = data.templates || [];
                        } catch (error) {
                            console.error('Failed to fetch templates:', error);
                            this.templates = [];
                        }
                    },

                    applyTemplate() {
                        if (!this.newTaskTemplateId) return;
                        const tmpl = this.templates.find(t => t.id === this.newTaskTemplateId);
                        if (!tmpl) return;
                        if (!this.newTaskInstructions.trim() && tmpl.instructions) {
                            this.newTaskInstructions = tmpl.instructions;
                        }
                        if (!this.newTaskTitle.trim() && tmpl.name) {
                            this.newTaskTitle = tmpl.name;
                        }
                        if (!this.newTaskType && tmpl.default_type) {
                            this.newTaskType = tmpl.default_type.toLowerCase();
                        }
                        if (!this.newTaskPriority && tmpl.default_priority > 0) {
                            if (tmpl.default_priority >= 70) this.newTaskPriority = 'high';
                            else if (tmpl.default_priority >= 40) this.newTaskPriority = 'medium';
                            else this.newTaskPriority = 'low';
                        }
                        if (!this.newTaskTags.trim() && tmpl.default_tags && tmpl.default_tags.length) {
                            this.newTaskTags = tmpl.default_tags.join(', ');
                        }
                    },

                    onProjectChange() {
                        // Reconnect SSE with new project filter
                        this.connectSSE();
                        this.fetchTemplates();
                    },

                    formatTime(iso) {
                        if (!iso) return '';
                        const d = new Date(iso);
                        const now = new Date();
                        const diff = now - d;
                        if (diff < 60000) return 'just now';
                        if (diff < 3600000) return Math.floor(diff / 60000) + 'm ago';
                        if (diff < 86400000) return Math.floor(diff / 3600000) + 'h ago';
                        if (diff < 604800000) return Math.floor(diff / 86400000) + 'd ago';
                        return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                    },

                    async fetchWorkdir() {
                        try {
                            const response = await fetch('/api/workdir');
                            const data = await response.json();
                            this.workdir = data.workdir;
                        } catch (error) {
                            console.error('Failed to fetch workdir:', error);
                        }
                    },

                    async updateWorkdir() {
                        const formData = new FormData();
                        formData.append('workdir', this.workdir);
                        await fetch('/set-workdir', {
                            method: 'POST',
                            body: formData
                        });
                    },

                    connectSSE() {
                        if (this.eventSource) {
                            this.eventSource.close();
                        }
                        this.sseReconnecting = true;
                        const url = this.currentProject ? '/events?project_id=' + encodeURIComponent(this.currentProject) : '/events';
                        this.eventSource = new EventSource(url);

                        this.eventSource.onopen = () => {
                            this.sseConnected = true;
                            this.sseReconnecting = false;
                        };

                        // SSE sends named "tasks" events
                        this.eventSource.addEventListener('tasks', (event) => {
                            try {
                                this.tasks = JSON.parse(event.data);
                            } catch (e) {
                                console.error('Failed to parse SSE task data:', e);
                            }
                        });

                        this.eventSource.onerror = () => {
                            this.sseConnected = false;
                            this.sseReconnecting = true;
                            this.eventSource.close();
                            // Reconnect after 3 seconds
                            setTimeout(() => this.connectSSE(), 3000);
                        };
                    },

                    openModal() {
                        this.showModal = true;
                        this.$nextTick(() => this.$refs.taskInput.focus());
                    },

                    copyInstructions() {
                        const text = this.$refs.instructionsText.textContent;
                        navigator.clipboard.writeText(text).then(() => {
                            this.copiedInstructions = true;
                            setTimeout(() => {
                                this.copiedInstructions = false;
                            }, 2000);
                        });
                    },

                    async createTask() {
                        if (!this.newTaskInstructions.trim()) return;

                        const payload = {
                            instructions: this.newTaskInstructions,
                            project_id: this.currentProject
                        };
                        if (this.newTaskTitle.trim()) payload.title = this.newTaskTitle.trim();
                        if (this.newTaskType) payload.type = this.newTaskType.toUpperCase();
                        if (this.newTaskPriority === 'high') payload.priority = 80;
                        else if (this.newTaskPriority === 'medium') payload.priority = 50;
                        else if (this.newTaskPriority === 'low') payload.priority = 20;
                        if (this.newTaskParentId) payload.parent_task_id = parseInt(this.newTaskParentId, 10);
                        if (this.newTaskTags.trim()) payload.tags = this.newTaskTags.split(',').map(t => t.trim()).filter(Boolean);
                        if (this.newTaskRequiresApproval) payload.requires_approval = true;
                        if (this.newTaskDependencies.length) payload.depends_on = this.newTaskDependencies.map(Number);
                        if (this.newTaskTemplateId) payload.template_id = this.newTaskTemplateId;

                        try {
                            const url = '/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/tasks';
                            const response = await fetch(url, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify(payload)
                            });
                            if (!response.ok) {
                                const data = await response.json().catch(() => ({}));
                                const msg = (data.error && data.error.message) || response.statusText;
                                this.showToast('Failed to create task: ' + msg, 'error');
                                return;
                            }
                        } catch (error) {
                            this.showToast('Failed to create task: ' + error.message, 'error');
                            return;
                        }

                        this.newTaskInstructions = '';
                        this.newTaskParentId = '';
                        this.newTaskTitle = '';
                        this.newTaskType = '';
                        this.newTaskPriority = '';
                        this.newTaskTags = '';
                        this.newTaskRequiresApproval = false;
                        this.newTaskDependencies = [];
                        this.newTaskTemplateId = '';
                        this.showModal = false;
                        this.showToast('Task created', 'success');
                    },

                    toggleExpand(taskId) {
                        const index = this.expandedTasks.indexOf(taskId);
                        if (index === -1) {
                            this.expandedTasks.push(taskId);
                        } else {
                            this.expandedTasks.splice(index, 1);
                        }
                    },

                    startDrag(task) {
                        this.draggingId = task.id;
                        this.draggingTask = task;
                    },

                    endDrag() {
                        this.draggingId = null;
                        this.draggingTask = null;
                        this.dragOverColumn = null;
                    },

                    async dropTask(newStatus) {
                        if (!this.draggingTask || this.draggingTask.status === newStatus) {
                            this.endDrag();
                            return;
                        }

                        const originalStatus = this.draggingTask.status;
                        // Optimistic update
                        this.draggingTask.status = newStatus;

                        try {
                            const response = await fetch('/api/v2/tasks/' + this.draggingTask.id, {
                                method: 'PATCH',
                                headers: {'Content-Type': 'application/json'},
                                body: JSON.stringify({status: newStatus})
                            });
                            if (!response.ok) {
                                this.draggingTask.status = originalStatus;
                            }
                        } catch (error) {
                            this.draggingTask.status = originalStatus;
                        }

                        this.endDrag();
                    },

                    async deleteTask(taskId) {
                        this.showConfirm('Are you sure you want to delete task #' + taskId + '?', async () => {
                            try {
                                const response = await fetch('/api/v2/tasks/' + taskId, {
                                    method: 'DELETE'
                                });
                                if (!response.ok) {
                                    const data = await response.json().catch(() => ({}));
                                    this.showToast('Failed to delete task: ' + (data.error?.message || response.statusText), 'error');
                                } else {
                                    this.showToast('Task #' + taskId + ' deleted', 'success');
                                }
                            } catch (error) {
                                this.showToast('Failed to delete task: ' + error.message, 'error');
                            }
                        });
                    },

                    async claimSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/claim', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ agent_id: 'manual' })
                            });
                            if (resp.ok) { this.showToast('Task claimed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Claim failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Claim failed: ' + e.message, 'error'); }
                    },

                    async completeSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/complete', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ output: 'Manually completed' })
                            });
                            if (resp.ok) { this.showToast('Task completed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Complete failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Complete failed: ' + e.message, 'error'); }
                    },

                    async failSelectedTask() {
                        if (!this.selectedTask) return;
                        try {
                            const resp = await fetch('/api/v2/tasks/' + this.selectedTask.id + '/fail', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ error: 'Manually failed' })
                            });
                            if (resp.ok) { this.showToast('Task failed', 'success'); this.selectedTask = null; }
                            else { const d = await resp.json().catch(() => ({})); this.showToast('Fail failed: ' + ((d.error && d.error.message) || resp.statusText), 'error'); }
                        } catch (e) { this.showToast('Fail failed: ' + e.message, 'error'); }
                    },

                    async duplicateSelectedTask() {
                        if (!this.selectedTask) return;
                        const t = this.selectedTask;
                        const payload = {
                            instructions: t.instructions,
                            project_id: this.currentProject,
                        };
                        if (t.title) payload.title = t.title + ' (copy)';
                        if (t.type) payload.type = t.type;
                        if (t.priority) payload.priority = t.priority;
                        if (t.tags) payload.tags = t.tags;
                        try {
                            const url = '/api/v2/projects/' + encodeURIComponent(this.currentProject) + '/tasks';
                            const resp = await fetch(url, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify(payload)
                            });
                            if (resp.ok) { this.showToast('Task duplicated', 'success'); }
                            else { this.showToast('Duplicate failed', 'error'); }
                        } catch (e) { this.showToast('Duplicate failed: ' + e.message, 'error'); }
                    }
                };
            }
        </script>
    </body>
</html>
`
