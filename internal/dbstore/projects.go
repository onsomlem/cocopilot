package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateProject(db *sql.DB, name, workdir string, settings map[string]interface{}) (*models.Project, error) {
	project := &models.Project{
		ID:        "proj_" + uuid.New().String(),
		Name:      name,
		Workdir:   workdir,
		Settings:  settings,
		CreatedAt: models.NowISO(),
	}

	settingsJSON, err := models.MarshalJSON(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO projects (id, name, workdir, settings_json, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, project.ID, project.Name, project.Workdir, settingsJSON, project.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	return project, nil
}

func GetProject(db *sql.DB, projectID string) (*models.Project, error) {
	var project models.Project
	var settingsJSON sql.NullString

	err := db.QueryRow(`
		SELECT id, name, workdir, settings_json, created_at
		FROM projects WHERE id = ?
	`, projectID).Scan(&project.ID, &project.Name, &project.Workdir, &settingsJSON, &project.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if settingsJSON.Valid && settingsJSON.String != "" {
		if err := models.UnmarshalJSON(settingsJSON.String, &project.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &project, nil
}

func ListProjects(db *sql.DB) ([]models.Project, error) {
	rows, err := db.Query(`
		SELECT id, name, workdir, settings_json, created_at
		FROM projects ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		var settingsJSON sql.NullString
		err := rows.Scan(&project.ID, &project.Name, &project.Workdir, &settingsJSON, &project.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		if settingsJSON.Valid && settingsJSON.String != "" {
			if err := models.UnmarshalJSON(settingsJSON.String, &project.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}
		projects = append(projects, project)
	}
	return projects, nil
}

func UpdateProject(db *sql.DB, projectID string, name *string, workdir *string, settings map[string]interface{}) (*models.Project, error) {
	project, err := GetProject(db, projectID)
	if err != nil {
		return nil, err
	}

	if name != nil {
		project.Name = *name
	}
	if workdir != nil {
		project.Workdir = *workdir
	}
	if settings != nil {
		project.Settings = settings
	}

	settingsJSON, err := models.MarshalJSON(project.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = db.Exec(`
		UPDATE projects SET name = ?, workdir = ?, settings_json = ?
		WHERE id = ?
	`, project.Name, project.Workdir, settingsJSON, projectID)

	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

func DeleteProject(db *sql.DB, projectID string) error {
	_, err := GetProject(db, projectID)
	if err != nil {
		return err
	}

	if projectID == "proj_default" {
		return fmt.Errorf("cannot delete default project")
	}

	_, err = db.Exec("DELETE FROM projects WHERE id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
