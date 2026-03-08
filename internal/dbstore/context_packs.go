package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateContextPack(db *sql.DB, projectID string, taskID int, summary string, contents map[string]interface{}) (*models.ContextPack, error) {
	pack := &models.ContextPack{
		ID:        "ctx_" + uuid.New().String(),
		ProjectID: projectID,
		TaskID:    taskID,
		Summary:   summary,
		Contents:  contents,
		CreatedAt: models.NowISO(),
	}

	contentsJSON, err := models.MarshalJSON(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contents: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO context_packs (id, project_id, task_id, summary, contents_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, pack.ID, pack.ProjectID, pack.TaskID, pack.Summary, contentsJSON, pack.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create context pack: %w", err)
	}
	return pack, nil
}

func GetContextPackByTaskID(db *sql.DB, taskID int) (*models.ContextPack, error) {
	var pack models.ContextPack
	var contentsJSON sql.NullString
	var stale int

	err := db.QueryRow(`
		SELECT id, project_id, task_id, summary, contents_json, created_at, COALESCE(stale, 0)
		FROM context_packs WHERE task_id = ? ORDER BY created_at DESC LIMIT 1
	`, taskID).Scan(&pack.ID, &pack.ProjectID, &pack.TaskID, &pack.Summary, &contentsJSON, &pack.CreatedAt, &stale)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get context pack: %w", err)
	}
	pack.Stale = stale != 0

	if contentsJSON.Valid && contentsJSON.String != "" {
		if err := models.UnmarshalJSON(contentsJSON.String, &pack.Contents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contents: %w", err)
		}
	}
	return &pack, nil
}

func GetContextPackByID(db *sql.DB, packID string) (*models.ContextPack, error) {
	var pack models.ContextPack
	var contentsJSON sql.NullString
	var stale int

	err := db.QueryRow(`
		SELECT id, project_id, task_id, summary, contents_json, created_at, COALESCE(stale, 0)
		FROM context_packs WHERE id = ?
	`, packID).Scan(&pack.ID, &pack.ProjectID, &pack.TaskID, &pack.Summary, &contentsJSON, &pack.CreatedAt, &stale)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("context pack not found: %s", packID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get context pack: %w", err)
	}
	pack.Stale = stale != 0

	if contentsJSON.Valid && contentsJSON.String != "" {
		if err := models.UnmarshalJSON(contentsJSON.String, &pack.Contents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contents: %w", err)
		}
	}
	return &pack, nil
}

func MarkContextPacksStale(db *sql.DB, projectID string) error {
	_, err := db.Exec(`UPDATE context_packs SET stale = 1 WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("failed to mark context packs stale: %w", err)
	}
	return nil
}

func RefreshContextPack(db *sql.DB, projectID string, taskID int, summary string, contents map[string]interface{}) (*models.ContextPack, error) {
	contentsJSON, err := models.MarshalJSON(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contents: %w", err)
	}
	now := models.NowISO()

	res, err := db.Exec(
		`UPDATE context_packs SET summary = ?, contents_json = ?, stale = 0, created_at = ? WHERE project_id = ? AND task_id = ?`,
		summary, contentsJSON, now, projectID, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh context pack: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows > 0 {
		return GetContextPackByTaskID(db, taskID)
	}

	return CreateContextPack(db, projectID, taskID, summary, contents)
}
