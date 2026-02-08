https://dganev.com/posts/2026-01-24-bypassing-copilot-requests/

# Agentic Task Queue

A web-based task queue server designed for orchestrating LLM agents. It provides a Kanban-style UI for managing tasks and a simple HTTP API for agents to poll for work and submit results.

## Prerequisites

- Go 1.21 or later
- SQLite3 (included via go-sqlite3)

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd theinf-loop

# Install dependencies
go mod download

# Build the server
go build -o task-server .
```

## Running the Server

```bash
# Run directly
go run main.go

# Or run the built binary
./task-server
```

The server starts on `http://0.0.0.0:8000`.

## Web Interface

Open `http://127.0.0.1:8000` in your browser to access the Kanban board UI where you can:

- Create new tasks
- View tasks in three columns: To Do, In Progress, Done
- Drag and drop tasks between columns
- Delete tasks
- Set the working directory for agents
- View task outputs

## API Endpoints

### Get Next Task

```
GET /task
```

Returns the next available task (oldest NOT_PICKED task first). Automatically marks the task as IN_PROGRESS.

**Response (task available):**
```
AVAILABLE TASK ID: 1
TASK_STATUS: IN_PROGRESS

Instructions:
<task instructions here>
```

**Response (no tasks):**
```
No tasks available.
Wait 15 secs for new instructions.
```

### Create Task

```
POST /create
Content-Type: application/x-www-form-urlencoded

instructions=<task instructions>&parent_task_id=<optional parent id>
```

**Response:**
```json
{"success": true, "task_id": 1}
```

### Save Task Output

```
POST /save
Content-Type: application/x-www-form-urlencoded

task_id=<id>&message=<output summary>
```

Marks the task as COMPLETE and stores the output.

### Update Task Status

```
POST /update-status
Content-Type: application/x-www-form-urlencoded

task_id=<id>&status=<NOT_PICKED|IN_PROGRESS|COMPLETE>
```

### Delete Task

```
POST /delete
Content-Type: application/x-www-form-urlencoded

task_id=<id>
```

### Get All Tasks (JSON)

```
GET /api/tasks
```

### Get/Set Working Directory

```
GET /api/workdir
POST /set-workdir (workdir=<path>)
```

### Server-Sent Events

```
GET /events
```

Real-time task updates for the web UI.

### Get Agent Instructions

```
GET /instructions
```

Returns the initial instructions for setting up an agent.

## Tutorial: Using with an LLM Agent

### Step 1: Start the Server

```bash
go run main.go
```

### Step 2: Set the Working Directory

Open `http://127.0.0.1:8000` and set the working directory where your agents should operate.

### Step 3: Create a Task

Click "New Task" and enter instructions for your agent:

```
Create a Python script that prints "Hello, World!"
```

### Step 4: Start Your Agent

Give your LLM agent these initial instructions:

```
See http://127.0.0.1:8000/instructions use curl to get the instructions and proceed with it. Do not stop when there are no new tasks!
```

### Step 5: Agent Workflow

The agent will:

1. Poll `GET /task` to receive work
2. Execute the task instructions
3. Submit results via `POST /save`:
   ```bash
   curl -X POST http://127.0.0.1:8000/save \
     -d "task_id=1" \
     -d "message=Created hello.py script that prints Hello World"
   ```
4. Wait 15 seconds
5. Poll `/task` again for the next task

### Step 6: Chain Tasks

Tasks can have parent-child relationships. When creating a task, select a parent task to build context chains. Child tasks automatically receive context from their parent's output.

## Task States

| Status | Description |
|--------|-------------|
| `NOT_PICKED` | Task is queued, waiting to be picked up |
| `IN_PROGRESS` | Task has been assigned to an agent |
| `COMPLETE` | Task finished, output saved |

## Database

Tasks are stored in `tasks.db` (SQLite) in the current directory. Delete this file to reset all tasks.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   Web Browser   │────▶│   Go Server     │
│  (Kanban UI)    │◀────│   :8000         │
└─────────────────┘     └────────┬────────┘
                                 │
┌─────────────────┐              │
│   LLM Agent     │──────────────┤
│  (polls /task)  │              │
└─────────────────┘              ▼
                        ┌─────────────────┐
                        │   SQLite DB     │
                        │   tasks.db      │
                        └─────────────────┘
```

## License

MIT
