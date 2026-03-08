package dbstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateMemory(db *sql.DB, projectID, scope, key string, value map[string]interface{}, sourceRefs []string) (*models.Memory, error) {
	memory := &models.Memory{
		ID:         "mem_" + uuid.New().String(),
		ProjectID:  projectID,
		Scope:      scope,
		Key:        key,
		Value:      value,
		SourceRefs: sourceRefs,
		CreatedAt:  models.NowISO(),
		UpdatedAt:  models.NowISO(),
	}

	valueJSON, err := models.MarshalJSON(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	sourceRefsJSON, err := models.MarshalJSON(sourceRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source_refs: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO memory (id, project_id, scope, key, value_json, source_refs_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, scope, key) 
		DO UPDATE SET value_json = ?, source_refs_json = ?, updated_at = ?
	`, memory.ID, memory.ProjectID, memory.Scope, memory.Key, valueJSON, sourceRefsJSON,
		memory.CreatedAt, memory.UpdatedAt, valueJSON, sourceRefsJSON, models.NowISO())

	if err != nil {
		return nil, fmt.Errorf("failed to create/update memory: %w", err)
	}
	return memory, nil
}

func GetMemory(db *sql.DB, projectID, scope, key string) (*models.Memory, error) {
	var memory models.Memory
	var valueJSON, sourceRefsJSON sql.NullString

	err := db.QueryRow(`
		SELECT id, project_id, scope, key, value_json, source_refs_json, created_at, updated_at
		FROM memory WHERE project_id = ? AND scope = ? AND key = ?
	`, projectID, scope, key).Scan(&memory.ID, &memory.ProjectID, &memory.Scope, &memory.Key,
		&valueJSON, &sourceRefsJSON, &memory.CreatedAt, &memory.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	if valueJSON.Valid && valueJSON.String != "" {
		if err := models.UnmarshalJSON(valueJSON.String, &memory.Value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}
	}
	if sourceRefsJSON.Valid && sourceRefsJSON.String != "" {
		if err := models.UnmarshalJSON(sourceRefsJSON.String, &memory.SourceRefs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source_refs: %w", err)
		}
	}
	return &memory, nil
}

func GetMemoriesByScope(db *sql.DB, projectID, scope string) ([]models.Memory, error) {
	rows, err := db.Query(`
		SELECT id, project_id, scope, key, value_json, source_refs_json, created_at, updated_at
		FROM memory WHERE project_id = ? AND scope = ? ORDER BY updated_at DESC
	`, projectID, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var memories []models.Memory
	for rows.Next() {
		var memory models.Memory
		var valueJSON, sourceRefsJSON sql.NullString
		err := rows.Scan(&memory.ID, &memory.ProjectID, &memory.Scope, &memory.Key,
			&valueJSON, &sourceRefsJSON, &memory.CreatedAt, &memory.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		if valueJSON.Valid && valueJSON.String != "" {
			if err := models.UnmarshalJSON(valueJSON.String, &memory.Value); err != nil {
				return nil, fmt.Errorf("failed to unmarshal value: %w", err)
			}
		}
		if sourceRefsJSON.Valid && sourceRefsJSON.String != "" {
			if err := models.UnmarshalJSON(sourceRefsJSON.String, &memory.SourceRefs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source_refs: %w", err)
			}
		}
		memories = append(memories, memory)
	}
	return memories, nil
}

func QueryMemories(db *sql.DB, projectID, scope, key, queryFilter string) ([]models.Memory, error) {
	filters := []string{"project_id = ?"}
	args := []interface{}{projectID}

	if strings.TrimSpace(scope) != "" {
		filters = append(filters, "scope = ?")
		args = append(args, scope)
	}
	if strings.TrimSpace(key) != "" {
		filters = append(filters, "key = ?")
		args = append(args, key)
	}
	if strings.TrimSpace(queryFilter) != "" {
		pattern := "%" + strings.ToLower(queryFilter) + "%"
		filters = append(filters, "(LOWER(key) LIKE ? OR LOWER(value_json) LIKE ?)")
		args = append(args, pattern, pattern)
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT id, project_id, scope, key, value_json, source_refs_json, created_at, updated_at
		FROM memory
	`)
	if len(filters) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(filters, " AND "))
	}
	query.WriteString(" ORDER BY updated_at DESC")

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var memories []models.Memory
	for rows.Next() {
		var memory models.Memory
		var valueJSON, sourceRefsJSON sql.NullString
		err := rows.Scan(&memory.ID, &memory.ProjectID, &memory.Scope, &memory.Key,
			&valueJSON, &sourceRefsJSON, &memory.CreatedAt, &memory.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		if valueJSON.Valid && valueJSON.String != "" {
			if err := models.UnmarshalJSON(valueJSON.String, &memory.Value); err != nil {
				return nil, fmt.Errorf("failed to unmarshal value: %w", err)
			}
		}
		if sourceRefsJSON.Valid && sourceRefsJSON.String != "" {
			if err := models.UnmarshalJSON(sourceRefsJSON.String, &memory.SourceRefs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source_refs: %w", err)
			}
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

func DeleteMemory(db *sql.DB, projectID, scope, key string) error {
	_, err := db.Exec("DELETE FROM memory WHERE project_id = ? AND scope = ? AND key = ?", projectID, scope, key)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}
	return nil
}
