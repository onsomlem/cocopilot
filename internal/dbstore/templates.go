package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateTaskTemplate(db *sql.DB, projectID, name string, description *string, instructions string, defaultType *string, defaultPriority int, defaultTags []string, defaultMetadata map[string]interface{}) (*models.TaskTemplate, error) {
	t := &models.TaskTemplate{
		ID:              "tmpl_" + uuid.New().String(),
		ProjectID:       projectID,
		Name:            name,
		Description:     description,
		Instructions:    instructions,
		DefaultType:     defaultType,
		DefaultPriority: defaultPriority,
		DefaultTags:     defaultTags,
		DefaultMetadata: defaultMetadata,
		CreatedAt:       models.NowISO(),
		UpdatedAt:       models.NowISO(),
	}

	tagsJSON, _ := models.MarshalJSON(defaultTags)
	metaJSON, _ := models.MarshalJSON(defaultMetadata)
	desc := sql.NullString{}
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}
	typeNull := sql.NullString{}
	if defaultType != nil {
		typeNull = sql.NullString{String: *defaultType, Valid: true}
	}

	_, err := db.Exec(`
		INSERT INTO task_templates (id, project_id, name, description, instructions, default_type, default_priority, default_tags, default_metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ProjectID, t.Name, desc, t.Instructions, typeNull, t.DefaultPriority, tagsJSON, metaJSON, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create task template: %w", err)
	}
	return t, nil
}

func GetTaskTemplate(db *sql.DB, templateID string) (*models.TaskTemplate, error) {
	var t models.TaskTemplate
	var desc, dtype, tagsJSON, metaJSON sql.NullString

	err := db.QueryRow(`
		SELECT id, project_id, name, description, instructions, default_type, default_priority, default_tags, default_metadata, created_at, updated_at
		FROM task_templates WHERE id = ?
	`, templateID).Scan(&t.ID, &t.ProjectID, &t.Name, &desc, &t.Instructions, &dtype, &t.DefaultPriority, &tagsJSON, &metaJSON, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if desc.Valid {
		t.Description = &desc.String
	}
	if dtype.Valid {
		t.DefaultType = &dtype.String
	}
	if tagsJSON.Valid && tagsJSON.String != "" {
		models.UnmarshalJSON(tagsJSON.String, &t.DefaultTags)
	}
	if metaJSON.Valid && metaJSON.String != "" {
		models.UnmarshalJSON(metaJSON.String, &t.DefaultMetadata)
	}
	return &t, nil
}

func ListTaskTemplates(db *sql.DB, projectID string) ([]models.TaskTemplate, error) {
	rows, err := db.Query(`
		SELECT id, project_id, name, description, instructions, default_type, default_priority, default_tags, default_metadata, created_at, updated_at
		FROM task_templates WHERE project_id = ? ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []models.TaskTemplate
	for rows.Next() {
		var t models.TaskTemplate
		var desc, dtype, tagsJSON, metaJSON sql.NullString
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Name, &desc, &t.Instructions, &dtype, &t.DefaultPriority, &tagsJSON, &metaJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		if desc.Valid {
			t.Description = &desc.String
		}
		if dtype.Valid {
			t.DefaultType = &dtype.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			models.UnmarshalJSON(tagsJSON.String, &t.DefaultTags)
		}
		if metaJSON.Valid && metaJSON.String != "" {
			models.UnmarshalJSON(metaJSON.String, &t.DefaultMetadata)
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func UpdateTaskTemplate(db *sql.DB, templateID string, name *string, description *string, instructions *string, defaultType *string, defaultPriority *int, defaultTags []string, defaultMetadata map[string]interface{}) (*models.TaskTemplate, error) {
	t, err := GetTaskTemplate(db, templateID)
	if err != nil {
		return nil, err
	}

	if name != nil {
		t.Name = *name
	}
	if description != nil {
		t.Description = description
	}
	if instructions != nil {
		t.Instructions = *instructions
	}
	if defaultType != nil {
		t.DefaultType = defaultType
	}
	if defaultPriority != nil {
		t.DefaultPriority = *defaultPriority
	}
	if defaultTags != nil {
		t.DefaultTags = defaultTags
	}
	if defaultMetadata != nil {
		t.DefaultMetadata = defaultMetadata
	}
	t.UpdatedAt = models.NowISO()

	tagsJSON, _ := models.MarshalJSON(t.DefaultTags)
	metaJSON, _ := models.MarshalJSON(t.DefaultMetadata)
	descNull := sql.NullString{}
	if t.Description != nil {
		descNull = sql.NullString{String: *t.Description, Valid: true}
	}
	typeNull := sql.NullString{}
	if t.DefaultType != nil {
		typeNull = sql.NullString{String: *t.DefaultType, Valid: true}
	}

	_, err = db.Exec(`
		UPDATE task_templates SET name = ?, description = ?, instructions = ?, default_type = ?, default_priority = ?, default_tags = ?, default_metadata = ?, updated_at = ?
		WHERE id = ?
	`, t.Name, descNull, t.Instructions, typeNull, t.DefaultPriority, tagsJSON, metaJSON, t.UpdatedAt, t.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}
	return t, nil
}

func DeleteTaskTemplate(db *sql.DB, templateID string) error {
	result, err := db.Exec(`DELETE FROM task_templates WHERE id = ?`, templateID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("template not found: %s", templateID)
	}
	return nil
}
