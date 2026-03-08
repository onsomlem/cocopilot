package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Export/Import — Project archive operations
// ============================================================================

// ProjectArchive represents a complete exported project with all entities.
type ProjectArchive struct {
	Version       string           `json:"version"`
	SchemaVersion int              `json:"schema_version"`
	Project       Project          `json:"project"`
	Tasks         []TaskV2         `json:"tasks"`
	Memories      []Memory         `json:"memories"`
	Policies      []Policy         `json:"policies"`
	Events        []Event          `json:"events,omitempty"`
}

// v2ProjectExportHandler handles GET /api/v2/projects/{projectId}/export
func v2ProjectExportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project ID required", nil)
		return
	}
	projectID := parts[0]

	project, err := GetProject(db, projectID)
	if err != nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Project not found", nil)
		return
	}

	// Gather tasks
	tasks, _, err := ListTasksV2(db, projectID, "", "", "", "", "", 10000, 0, "created_at", "asc")
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to list tasks: "+err.Error(), nil)
		return
	}
	if tasks == nil {
		tasks = []TaskV2{}
	}

	// Gather memories
	memories, err := QueryMemories(db, projectID, "", "", "")
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to list memories: "+err.Error(), nil)
		return
	}
	if memories == nil {
		memories = []Memory{}
	}

	// Gather policies
	policies, _, err := ListPoliciesByProject(db, projectID, nil, 1000, 0, "created_at", "asc")
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to list policies: "+err.Error(), nil)
		return
	}
	if policies == nil {
		policies = []Policy{}
	}

	// Gather recent events (limited)
	events, err := GetEventsByProjectID(db, projectID, 500)
	if err != nil {
		events = []Event{}
	}
	if events == nil {
		events = []Event{}
	}

	archive := ProjectArchive{
		Version:       "1.0",
		SchemaVersion: getCurrentSchemaVersion(),
		Project:       *project,
		Tasks:         tasks,
		Memories:      memories,
		Policies:      policies,
		Events:        events,
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_export.json"`, projectID))
	writeV2JSON(w, http.StatusOK, archive)
}

// v2ProjectImportHandler handles POST /api/v2/projects/import
func v2ProjectImportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var archive ProjectArchive
	if err := json.NewDecoder(r.Body).Decode(&archive); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body: "+err.Error(), nil)
		return
	}

	if archive.Project.Name == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project name is required in archive", nil)
		return
	}

	// Validate schema compatibility: if the archive declares a schema version,
	// it must not exceed the current server's schema version.
	if archive.SchemaVersion > 0 {
		currentSchema := getCurrentSchemaVersion()
		if archive.SchemaVersion > currentSchema {
			writeV2Error(w, http.StatusBadRequest, "SCHEMA_INCOMPATIBLE",
				fmt.Sprintf("Archive schema version %d is newer than server schema version %d. Upgrade the server first.",
					archive.SchemaVersion, currentSchema),
				map[string]interface{}{
					"archive_schema_version": archive.SchemaVersion,
					"server_schema_version":  currentSchema,
				})
			return
		}
	}

	// Create new project with a fresh ID
	newProjectID := "proj_" + uuid.New().String()
	workdir := archive.Project.Workdir
	if workdir == "" {
		workdir = "/tmp"
	}

	newProject, err := CreateProject(db, archive.Project.Name+" (imported)", workdir, archive.Project.Settings)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to create project: "+err.Error(), nil)
		return
	}
	newProjectID = newProject.ID

	// Track old task ID -> new task ID mapping
	taskIDMap := make(map[int]int)
	tasksImported := 0

	for _, task := range archive.Tasks {
		oldID := task.ID
		var parentID *int
		if task.ParentTaskID != nil {
			if newParent, ok := taskIDMap[*task.ParentTaskID]; ok {
				parentID = &newParent
			}
		}

		newTask, err := CreateTaskV2WithMeta(db, task.Instructions, newProjectID, parentID,
			task.Title, &task.Type, &task.Priority, task.Tags)
		if err != nil {
			continue
		}
		taskIDMap[oldID] = newTask.ID
		tasksImported++
	}

	// Import memories
	memoriesImported := 0
	for _, mem := range archive.Memories {
		_, err := CreateMemory(db, newProjectID, mem.Scope, mem.Key, mem.Value, mem.SourceRefs)
		if err == nil {
			memoriesImported++
		}
	}

	// Import policies
	policiesImported := 0
	for _, pol := range archive.Policies {
		_, err := CreatePolicy(db, newProjectID, pol.Name, pol.Description, pol.Rules, pol.Enabled)
		if err == nil {
			policiesImported++
		}
	}

	CreateEvent(db, newProjectID, "project.imported", "project", newProjectID, map[string]interface{}{
		"source_project_id": archive.Project.ID,
		"tasks_imported":    tasksImported,
		"memories_imported": memoriesImported,
		"policies_imported": policiesImported,
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"project":           newProject,
		"tasks_imported":    tasksImported,
		"memories_imported": memoriesImported,
		"policies_imported": policiesImported,
		"task_id_map":       taskIDMap,
	})
}
