package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

// CreatePromptTemplate inserts a new prompt template.
func CreatePromptTemplate(db *sql.DB, projectID, role, name, systemPrompt, userTemplate string, description, outputSchema *string) (*models.PromptTemplate, error) {
	id := "pt_" + uuid.New().String()
	now := models.NowISO()

	// Find next version for this project+role
	var maxVersion int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM prompt_templates WHERE project_id = ? AND role = ?`, projectID, role).Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get max version: %w", err)
	}
	version := maxVersion + 1

	_, err = db.Exec(`INSERT INTO prompt_templates (id, project_id, role, version, name, description, system_prompt, user_template, output_schema, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		id, projectID, role, version, name, description, systemPrompt, userTemplate, outputSchema, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt template: %w", err)
	}

	// Deactivate older versions of same role
	_, _ = db.Exec(`UPDATE prompt_templates SET is_active = 0, updated_at = ? WHERE project_id = ? AND role = ? AND version < ?`, now, projectID, role, version)

	return GetPromptTemplate(db, id)
}

// GetPromptTemplate retrieves a prompt template by ID.
func GetPromptTemplate(db *sql.DB, id string) (*models.PromptTemplate, error) {
	row := db.QueryRow(`SELECT id, project_id, role, version, name, description, system_prompt, user_template, output_schema, is_active, created_at, updated_at
		FROM prompt_templates WHERE id = ?`, id)

	var pt models.PromptTemplate
	var isActive int
	var desc, schema sql.NullString
	err := row.Scan(&pt.ID, &pt.ProjectID, &pt.Role, &pt.Version, &pt.Name, &desc, &pt.SystemPrompt, &pt.UserTemplate, &schema, &isActive, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		return nil, err
	}
	pt.IsActive = isActive == 1
	if desc.Valid {
		pt.Description = &desc.String
	}
	if schema.Valid {
		pt.OutputSchema = &schema.String
	}
	return &pt, nil
}

// GetActivePromptByRole retrieves the active version of a prompt template for a given role.
func GetActivePromptByRole(db *sql.DB, projectID, role string) (*models.PromptTemplate, error) {
	row := db.QueryRow(`SELECT id, project_id, role, version, name, description, system_prompt, user_template, output_schema, is_active, created_at, updated_at
		FROM prompt_templates WHERE project_id = ? AND role = ? AND is_active = 1`, projectID, role)

	var pt models.PromptTemplate
	var isActive int
	var desc, schema sql.NullString
	err := row.Scan(&pt.ID, &pt.ProjectID, &pt.Role, &pt.Version, &pt.Name, &desc, &pt.SystemPrompt, &pt.UserTemplate, &schema, &isActive, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		return nil, err
	}
	pt.IsActive = isActive == 1
	if desc.Valid {
		pt.Description = &desc.String
	}
	if schema.Valid {
		pt.OutputSchema = &schema.String
	}
	return &pt, nil
}

// ListPromptTemplates lists prompt templates for a project, optionally filtered by role.
func ListPromptTemplates(db *sql.DB, projectID, role string, activeOnly bool) ([]models.PromptTemplate, error) {
	query := `SELECT id, project_id, role, version, name, description, system_prompt, user_template, output_schema, is_active, created_at, updated_at
		FROM prompt_templates WHERE project_id = ?`
	args := []interface{}{projectID}

	if role != "" {
		query += " AND role = ?"
		args = append(args, role)
	}
	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY role, version DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.PromptTemplate
	for rows.Next() {
		var pt models.PromptTemplate
		var isActive int
		var desc, schema sql.NullString
		if err := rows.Scan(&pt.ID, &pt.ProjectID, &pt.Role, &pt.Version, &pt.Name, &desc, &pt.SystemPrompt, &pt.UserTemplate, &schema, &isActive, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
			return nil, err
		}
		pt.IsActive = isActive == 1
		if desc.Valid {
			pt.Description = &desc.String
		}
		if schema.Valid {
			pt.OutputSchema = &schema.String
		}
		templates = append(templates, pt)
	}
	return templates, nil
}

// UpdatePromptTemplate updates a prompt template's content fields.
func UpdatePromptTemplate(db *sql.DB, pt *models.PromptTemplate) error {
	now := models.NowISO()
	pt.UpdatedAt = now
	_, err := db.Exec(`UPDATE prompt_templates SET name = ?, description = ?, system_prompt = ?, user_template = ?, output_schema = ?, updated_at = ? WHERE id = ?`,
		pt.Name, pt.Description, pt.SystemPrompt, pt.UserTemplate, pt.OutputSchema, now, pt.ID)
	return err
}

// ActivatePromptVersion sets a specific version as active and deactivates all others for the same role.
func ActivatePromptVersion(db *sql.DB, projectID, role string, version int) error {
	now := models.NowISO()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE prompt_templates SET is_active = 0, updated_at = ? WHERE project_id = ? AND role = ?`, now, projectID, role)
	if err != nil {
		return err
	}

	result, err := tx.Exec(`UPDATE prompt_templates SET is_active = 1, updated_at = ? WHERE project_id = ? AND role = ? AND version = ?`, now, projectID, role, version)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("version %d not found for role '%s'", version, role)
	}

	return tx.Commit()
}

// DeletePromptTemplate deletes a prompt template by ID.
func DeletePromptTemplate(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM prompt_templates WHERE id = ?`, id)
	return err
}
