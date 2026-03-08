// FastAPI Task Queue Web Server for Agentic LLM Loop - Go Version
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db         *sql.DB
	sseClients = make([]chan string, 0)
	sseMutex   sync.RWMutex
	workdir    = "/tmp"
	workdirMu  sync.RWMutex
)

func getInstructions() string {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	return `Those are your general instructions, this is an orchestrator tool for running subagents.

IMPORTANT - WORKING DIRECTORY:
All subagents MUST ALWAYS work in the following directory:
` + "`" + wd + "`" + `

Before doing any work, the subagent must first change to this directory:
` + "```" + `
cd ` + wd + `
` + "```" + `

Your task is to follow the instructions at the ` + "`/task`" + ` endpoint.
For each instruction, you will start a sub agent using the ` + "`runSubagent`" + ` tool.
The subagent must know what task ID is given, so that he can issue a request later on.

At the end of each sub agent, the sub agent must produce a summarized output
of what it has done in a few points, the sub agent then will then issue this request using curl:

` + "```" + `
POST http://127.0.0.1:8001/save
Content-Type: application/x-www-form-urlencoded
task_id=<task_id>&message=<summarized_output_here>
` + "```" + `

After that the subagent would return the control back to the main orchestrator, which will poll
the ` + "`/task`" + ` endpoint after 15 seconds for new instructions. Use bash command to wait it out and
then issue a new request towards ` + "`/task`" + `. If there is a new task at hand follow the instructions
in the beginning again.

Important Note:

Always use ` + "`curl`" + ` to do the web requests.
`
}

type TaskStatus string

const (
	StatusNotPicked  TaskStatus = "NOT_PICKED"
	StatusInProgress TaskStatus = "IN_PROGRESS"
	StatusComplete   TaskStatus = "COMPLETE"
)

type Task struct {
	ID           int        `json:"id"`
	Instructions string     `json:"instructions"`
	Status       TaskStatus `json:"status"`
	Output       *string    `json:"output"`
	ParentTaskID *int       `json:"parent_task_id"`
	CreatedAt    string     `json:"created_at"`
}

func initDB() error {
	var err error
	dbPath := filepath.Join(".", "tasks.db")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instructions TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'NOT_PICKED',
			output TEXT,
			parent_task_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
		)
	`)
	if err != nil {
		return err
	}

	// Migration for existing DB - try to add column, ignore if exists
	_, _ = db.Exec("ALTER TABLE tasks ADD COLUMN parent_task_id INTEGER")

	return nil
}

func broadcastUpdate() {
	sseMutex.RLock()
	defer sseMutex.RUnlock()
	for _, client := range sseClients {
		select {
		case client <- "update":
		default:
			// Client buffer full, skip
		}
	}
}

func getAllTasksJSON() ([]Task, error) {
	rows, err := db.Query("SELECT id, instructions, status, output, parent_task_id, created_at FROM tasks ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		var output sql.NullString
		var parentID sql.NullInt64
		err := rows.Scan(&t.ID, &t.Instructions, &t.Status, &output, &parentID, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		if output.Valid {
			outputStr := strings.ReplaceAll(output.String, "\\n", "\n")
			t.Output = &outputStr
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			t.ParentTaskID = &pid
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func apiTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := getAllTasksJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan string, 10)
	sseMutex.Lock()
	sseClients = append(sseClients, clientChan)
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		for i, c := range sseClients {
			if c == clientChan {
				sseClients = append(sseClients[:i], sseClients[i+1:]...)
				break
			}
		}
		sseMutex.Unlock()
		close(clientChan)
	}()

	// Send initial data
	tasks, err := getAllTasksJSON()
	if err == nil {
		data, _ := json.Marshal(tasks)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-clientChan:
			tasks, err := getAllTasksJSON()
			if err == nil {
				data, _ := json.Marshal(tasks)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		case <-time.After(30 * time.Second):
			// Keep-alive ping
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	contextDepthStr := r.URL.Query().Get("context_depth")
	contextDepth := 3
	if contextDepthStr != "" {
		if d, err := strconv.Atoi(contextDepthStr); err == nil {
			contextDepth = d
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// First, look for a NOT_PICKED task (oldest first to process in order)
	var taskID int
	var status TaskStatus
	var instructions string
	var parentTaskID sql.NullInt64

	err := db.QueryRow("SELECT id, status, instructions, parent_task_id FROM tasks WHERE status = ? ORDER BY id ASC LIMIT 1", StatusNotPicked).
		Scan(&taskID, &status, &instructions, &parentTaskID)

	if err == sql.ErrNoRows {
		// No NOT_PICKED tasks, check if there's an IN_PROGRESS task
		err = db.QueryRow("SELECT id, status, instructions, parent_task_id FROM tasks WHERE status = ? ORDER BY id ASC LIMIT 1", StatusInProgress).
			Scan(&taskID, &status, &instructions, &parentTaskID)

		if err == sql.ErrNoRows {
			// No pending or in-progress tasks
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Return the in-progress task info
	}
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If task is NOT_PICKED, mark as IN_PROGRESS
	if status == StatusNotPicked {
		_, err = db.Exec("UPDATE tasks SET status = ? WHERE id = ?", StatusInProgress, taskID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		status = StatusInProgress
		go broadcastUpdate()
	}

	// Build context from ancestors
	var contextParts []string
	currentParentID := parentTaskID
	depth := 0

	for currentParentID.Valid && depth < contextDepth {
		var parentID int
		var parentOutput sql.NullString
		var nextParentID sql.NullInt64

		err := db.QueryRow("SELECT id, output, parent_task_id FROM tasks WHERE id = ?", currentParentID.Int64).
			Scan(&parentID, &parentOutput, &nextParentID)
		if err != nil {
			break
		}

		if parentOutput.Valid && parentOutput.String != "" {
			contextParts = append([]string{fmt.Sprintf("=== Task #%d Output ===\n%s", parentID, parentOutput.String)}, contextParts...)
		}

		currentParentID = nextParentID
		depth++
	}

	contextBlock := ""
	if len(contextParts) > 0 {
		contextBlock = "CONTEXT FROM PREVIOUS TASKS:\n" + strings.Join(contextParts, "\n\n") + "\n\n---\n\n"
	}

	fmt.Fprintf(w, "AVAILABLE TASK ID: %d\nTASK_STATUS: %s\n\n%sInstructions:\n%s", taskID, status, contextBlock, instructions)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	message := r.FormValue("message")

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		http.Error(w, "Invalid task_id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Check if task exists
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Task %d not found.", taskID)
		return
	}

	// Update task with output and mark as complete
	_, err = db.Exec("UPDATE tasks SET output = ?, status = ? WHERE id = ?", message, StatusComplete, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go broadcastUpdate()

	fmt.Fprintf(w, "Task %d saved and marked as COMPLETE.", taskID)
}

func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	status := r.FormValue("status")

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task_id"})
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		string(StatusNotPicked):  true,
		string(StatusInProgress): true,
		string(StatusComplete):   true,
	}
	if !validStatuses[status] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid status"})
		return
	}

	// Check if task exists
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	_, err = db.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	broadcastUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		// Fall back to regular form parsing
		err = r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
	}

	instructions := r.FormValue("instructions")
	parentTaskIDStr := r.FormValue("parent_task_id")

	var parentTaskID *int
	if parentTaskIDStr != "" {
		if id, err := strconv.Atoi(parentTaskIDStr); err == nil {
			parentTaskID = &id
		}
	}

	var result sql.Result
	if parentTaskID != nil {
		result, err = db.Exec("INSERT INTO tasks (instructions, status, parent_task_id) VALUES (?, ?, ?)",
			instructions, StatusNotPicked, *parentTaskID)
	} else {
		result, err = db.Exec("INSERT INTO tasks (instructions, status) VALUES (?, ?)",
			instructions, StatusNotPicked)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	taskID, _ := result.LastInsertId()

	broadcastUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "task_id": taskID})
}

func instructionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusSeeOther)
	fmt.Fprint(w, getInstructions())
}

func getWorkdirHandler(w http.ResponseWriter, r *http.Request) {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"workdir": wd})
}

func setWorkdirHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	newWorkdir := r.FormValue("workdir")
	if newWorkdir == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workdir is required"})
		return
	}

	workdirMu.Lock()
	workdir = newWorkdir
	workdirMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "workdir": newWorkdir})
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task_id"})
		return
	}

	// Check if task exists
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	// Delete the task
	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	broadcastUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlTemplate)
}

const htmlTemplate = `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Agentic Task Queue - Kanban</title>
        <script defer src="https://cdn.jsdelivr.net/npm/@alpinejs/collapse@3.x.x/dist/cdn.min.js"></script>
        <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
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
                            <label>Parent Task (optional)</label>
                            <select x-model="newTaskParentId">
                                <option value="">No parent (standalone task)</option>
                                <template x-for="task in tasks" :key="task.id">
                                    <option :value="task.id" x-text="'#' + task.id + ': ' + task.instructions.substring(0, 50) + (task.instructions.length > 50 ? '...' : '')"></option>
                                </template>
                            </select>
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
                            See http://127.0.0.1:8001/instructions use curl to get the instructions and proceed with it. Do not stop when there are no new tasks!
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
            <span class="header-title">Agentic Task Queue</span>
            <div class="header-actions">
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
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="task.created_at"></span>
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
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="todoTasks.length === 0">No tasks</div>
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
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="task.created_at"></span>
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
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="progressTasks.length === 0">No tasks</div>
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
                             draggable="true"
                             :class="{ 'dragging': draggingId === task.id }"
                             :data-depth="task.depth || 0"
                             @dragstart="startDrag(task)"
                             @dragend="endDrag">
                            <div class="card-header" @click="toggleExpand(task.id)">
                                <div class="card-left">
                                    <span class="card-id" x-text="'#' + task.id"></span>
                                    <span class="parent-badge" x-show="task.parent_task_id">
                                        <svg viewBox="0 0 16 16" fill="currentColor">
                                            <path d="M11 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8z" opacity=".3"/>
                                            <path d="M8 4a4 4 0 1 1-8 0 4 4 0 0 1 8 0z"/>
                                        </svg>
                                        from #<span x-text="task.parent_task_id"></span>
                                    </span>
                                    <span class="card-preview" x-text="task.instructions.substring(0, 40) + (task.instructions.length > 40 ? '...' : '')" x-show="!expandedTasks.includes(task.id) && !task.parent_task_id"></span>
                                </div>
                                <div class="card-right">
                                    <span class="card-time" x-text="task.created_at"></span>
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
                            <div class="card-body" x-show="expandedTasks.includes(task.id)" x-collapse>
                                <pre class="card-instructions" x-text="task.instructions"></pre>
                                <div class="card-output" x-show="task.output">
                                    <div class="card-output-label">Output</div>
                                    <pre class="card-output-text" x-text="task.output"></pre>
                                </div>
                            </div>
                        </div>
                    </template>
                    <div class="empty-column" x-show="doneTasks.length === 0">No tasks</div>
                </div>
            </div>
        </div>

        <footer class="footer">
            Created by <a href="https://dganev.com" target="_blank" rel="noopener">syl</a>
        </footer>

        <script>
            function kanbanApp() {
                return {
                    tasks: [],
                    expandedTasks: [],
                    showModal: false,
                    showInstructionsModal: false,
                    copiedInstructions: false,
                    newTaskInstructions: '',
                    newTaskParentId: '',
                    draggingId: null,
                    draggingTask: null,
                    dragOverColumn: null,
                    eventSource: null,
                    workdir: '/tmp',

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
                        this.connectSSE();
                        this.fetchWorkdir();
                    },

                    async fetchWorkdir() {
                        const response = await fetch('/api/workdir');
                        const data = await response.json();
                        this.workdir = data.workdir;
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
                        this.eventSource = new EventSource('/events');

                        this.eventSource.onmessage = (event) => {
                            this.tasks = JSON.parse(event.data);
                        };

                        this.eventSource.onerror = () => {
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

                        const formData = new FormData();
                        formData.append('instructions', this.newTaskInstructions);
                        if (this.newTaskParentId) {
                            formData.append('parent_task_id', this.newTaskParentId);
                        }

                        await fetch('/create', {
                            method: 'POST',
                            body: formData
                        });

                        this.newTaskInstructions = '';
                        this.newTaskParentId = '';
                        this.showModal = false;
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

                        const formData = new FormData();
                        formData.append('task_id', this.draggingTask.id);
                        formData.append('status', newStatus);

                        // Optimistic update
                        this.draggingTask.status = newStatus;

                        await fetch('/update-status', {
                            method: 'POST',
                            body: formData
                        });

                        this.endDrag();
                    },

                    async deleteTask(taskId) {
                        if (!confirm('Are you sure you want to delete this task?')) {
                            return;
                        }

                        const formData = new FormData();
                        formData.append('task_id', taskId);

                        await fetch('/delete', {
                            method: 'POST',
                            body: formData
                        });
                    }
                };
            }
        </script>
    </body>
    </html>
`

func main() {
	err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/tasks", apiTasksHandler)
	http.HandleFunc("/events", eventsHandler)
	http.HandleFunc("/task", getTaskHandler)
	http.HandleFunc("/save", saveHandler)
	http.HandleFunc("/update-status", updateStatusHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/instructions", instructionsHandler)
	http.HandleFunc("/api/workdir", getWorkdirHandler)
	http.HandleFunc("/set-workdir", setWorkdirHandler)
	http.HandleFunc("/delete", deleteTaskHandler)

	log.Println("Starting Agentic Task Queue server on http://0.0.0.0:8001")
	log.Fatal(http.ListenAndServe("0.0.0.0:8001", nil))
}
