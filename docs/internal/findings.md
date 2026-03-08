cocopilot\findings.md
```

# Project Findings: Agentic Task Queue

## Overview
This project is a **web-based task queue server** designed to orchestrate **LLM (Large Language Model) agents**. It provides a **Kanban-style UI** for task management and a **RESTful HTTP API** for agents to interact with the system. The server is implemented in **Go** and uses **SQLite** for task storage.

---

## Key Features

### 1. Web Interface
The project includes a web interface accessible at `http://127.0.0.1:8080`. This interface provides the following functionalities:
- **Task Management**:
  - Create new tasks.
  - View tasks in three columns: `To Do`, `In Progress`, and `Done`.
  - Drag and drop tasks between columns.
  - Delete tasks.
  - View task outputs.
- **Working Directory Management**:
  - Set the working directory for agents.
  - View the current working directory.

### 2. API Endpoints
The server exposes several endpoints for interacting with tasks and managing the system:

| Endpoint                  | Method | Description                                                                 |
|---------------------------|--------|-----------------------------------------------------------------------------|
| `/task`                   | GET    | Fetches the next available task and marks it as `IN_PROGRESS`.             |
| `/create`                 | POST   | Creates a new task with optional parent task ID.                           |
| `/save`                   | POST   | Marks a task as `COMPLETE` and stores the output.                          |
| `/update-status`          | POST   | Updates the status of a task.                                              |
| `/delete`                 | POST   | Deletes a task.                                                            |
| `/api/tasks`              | GET    | Fetches all tasks in JSON format.                                          |
| `/api/workdir`            | GET    | Retrieves the current working directory.                                   |
| `/set-workdir`            | POST   | Sets a new working directory.                                              |
| `/events`                 | GET    | Provides real-time task updates for the web UI using Server-Sent Events.   |
| `/instructions`           | GET    | Returns initial instructions for setting up an agent.                      |

---

## Database
The project uses an **SQLite database** (`tasks.db`) to store task information. The database schema is as follows:

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    instructions TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'NOT_PICKED',
    output TEXT,
    parent_task_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
);
```

### Task States
Tasks can have the following statuses:
- **`NOT_PICKED`**: Task is queued and waiting to be picked up.
- **`IN_PROGRESS`**: Task has been assigned to an agent.
- **`COMPLETE`**: Task is finished, and the output is saved.

---

## Agent Workflow
The project is designed to work with LLM agents. The typical workflow for an agent is as follows:
1. **Poll for Tasks**:
   - The agent sends a `GET` request to `/task` to fetch the next available task.
   - If a task is available, it is marked as `IN_PROGRESS` and returned to the agent.
   - If no tasks are available, the agent is instructed to wait for 15 seconds and try again.

2. **Execute Task**:
   - The agent performs the task based on the instructions provided.

3. **Submit Results**:
   - The agent sends a `POST` request to `/save` with the task ID and a summary of the output.
   - The task is marked as `COMPLETE`, and the output is stored in the database.

4. **Repeat**:
   - The agent waits for 15 seconds and polls `/task` again for the next task.

---

## Real-Time Updates
The project uses **Server-Sent Events (SSE)** to provide real-time updates to connected clients. The `/events` endpoint streams updates whenever tasks are created, updated, or deleted. This ensures that the web UI remains synchronized with the server.

---

## Code Structure

### Main File (`main.go`)
The `main.go` file contains the core implementation of the server. Below are the key components:

#### Initialization
- **`initDB`**:
  - Initializes the SQLite database.
  - Creates the `tasks` table if it doesn't exist.
  - Adds a `parent_task_id` column to support task hierarchies.

#### Task Management
- **`createHandler`**:
  - Handles task creation.
  - Supports optional parent-child relationships between tasks.
- **`getTaskHandler`**:
  - Fetches the next available task and marks it as `IN_PROGRESS`.
  - Builds a context block from parent tasks for hierarchical task management.
- **`saveHandler`**:
  - Saves task output and marks the task as `COMPLETE`.
- **`updateStatusHandler`**:
  - Updates the status of a task.
- **`deleteTaskHandler`**:
  - Deletes a task.

#### API Handlers
- **`apiTasksHandler`**:
  - Returns all tasks in JSON format.
- **`getWorkdirHandler`**:
  - Retrieves the current working directory.
- **`setWorkdirHandler`**:
  - Updates the working directory.

#### Real-Time Updates
- **`eventsHandler`**:
  - Handles Server-Sent Events (SSE) for real-time task updates.
- **`broadcastUpdate`**:
  - Notifies all connected clients of task updates.

#### Web Interface
- **`indexHandler`**:
  - Serves the HTML for the Kanban-style UI.

---

## Web Interface
The web interface is implemented using HTML, CSS, and JavaScript. It provides a user-friendly Kanban board for managing tasks. Key features include:
- Drag-and-drop functionality for moving tasks between columns.
- Real-time updates using SSE.
- Modals for creating new tasks and viewing initial instructions.
- Input for setting the working directory.

---

## Dependencies
The project uses the following Go modules:
- **`modernc.org/sqlite`**: SQLite database driver.
- **`github.com/dustin/go-humanize`**: Utility for human-readable formatting.
- **`github.com/google/uuid`**: UUID generation.
- **`github.com/hashicorp/golang-lru/v2`**: LRU cache implementation.
- **`github.com/ncruces/go-strftime`**: Date formatting.
- **`golang.org/x/sys`**: Low-level OS interactions.

---

## How to Run the Project

### Prerequisites
- **Go 1.21 or later**
- **SQLite3** (included via `go-sqlite3`)

### Installation
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd theinf-loop
   ```
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Build the server:
   ```bash
   go build -o task-server .
   ```

### Running the Server
1. Run directly:
   ```bash
   go run main.go
   ```
2. Or run the built binary:
   ```bash
   ./task-server
   ```

The server starts on `http://0.0.0.0:8080`.

---

## Architecture
The system architecture is as follows:

```
┌───────────────┐     ┌───────────────┐
│   Web Browser │◀───▶│   Go Server   │
│  (Kanban UI)  │     │   :8080       │
└───────────────┘     └───────┬───────┘
                               │
                               ▼
                        ┌───────────────┐
                        │   SQLite DB   │
                        │   tasks.db    │
                        └───────────────┘
```

---

## Conclusion
This project is a robust and extensible task queue server for orchestrating LLM agents. It combines a user-friendly web interface, a powerful API, and real-time updates to provide a seamless experience for managing tasks. The use of Go and SQLite ensures high performance and reliability.

If you have any questions or need further assistance, feel free to ask!