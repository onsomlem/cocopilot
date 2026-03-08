package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func UpsertRepoFile(db *sql.DB, file models.RepoFile) (*models.RepoFile, error) {
	now := models.NowISO()
	if file.ID == "" {
		file.ID = "rf_" + uuid.New().String()
	}
	file.UpdatedAt = now

	metadataJSON, err := models.MarshalJSON(file.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO repo_files (id, project_id, path, content_hash, size_bytes, language, last_modified, created_at, updated_at, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, path) DO UPDATE SET
			content_hash = excluded.content_hash,
			size_bytes = excluded.size_bytes,
			language = excluded.language,
			last_modified = excluded.last_modified,
			updated_at = excluded.updated_at,
			metadata_json = excluded.metadata_json
	`, file.ID, file.ProjectID, file.Path,
		models.NullString(file.ContentHash),
		ptrInt64ToNullInt64(file.SizeBytes),
		models.NullString(file.Language),
		models.NullString(file.LastModified),
		now, now, metadataJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert repo file: %w", err)
	}

	return GetRepoFile(db, file.ProjectID, file.Path)
}

func GetRepoFile(db *sql.DB, projectID, path string) (*models.RepoFile, error) {
	var f models.RepoFile
	var contentHash, language, lastModified sql.NullString
	var sizeBytes sql.NullInt64
	var metadataJSON sql.NullString

	err := db.QueryRow(`
		SELECT id, project_id, path, content_hash, size_bytes, language, last_modified, created_at, updated_at, metadata_json
		FROM repo_files WHERE project_id = ? AND path = ?
	`, projectID, path).Scan(
		&f.ID, &f.ProjectID, &f.Path, &contentHash, &sizeBytes, &language, &lastModified, &f.CreatedAt, &f.UpdatedAt, &metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repo file not found: %s/%s", projectID, path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repo file: %w", err)
	}

	f.ContentHash = models.PtrString(contentHash)
	f.SizeBytes = models.PtrInt64(sizeBytes)
	f.Language = models.PtrString(language)
	f.LastModified = models.PtrString(lastModified)
	if metadataJSON.Valid && metadataJSON.String != "" {
		var meta map[string]interface{}
		if err := models.UnmarshalJSON(metadataJSON.String, &meta); err == nil {
			f.Metadata = meta
		}
	}

	return &f, nil
}

func ListRepoFiles(db *sql.DB, projectID string, opts models.ListRepoFilesOpts) ([]models.RepoFile, int, error) {
	countQuery := "SELECT COUNT(*) FROM repo_files WHERE project_id = ?"
	args := []interface{}{projectID}

	if opts.Language != nil {
		countQuery += " AND language = ?"
		args = append(args, *opts.Language)
	}

	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count repo files: %w", err)
	}

	query := "SELECT id, project_id, path, content_hash, size_bytes, language, last_modified, created_at, updated_at, metadata_json FROM repo_files WHERE project_id = ?"
	listArgs := []interface{}{projectID}

	if opts.Language != nil {
		query += " AND language = ?"
		listArgs = append(listArgs, *opts.Language)
	}

	query += " ORDER BY path ASC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := db.Query(query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list repo files: %w", err)
	}
	defer rows.Close()

	var files []models.RepoFile
	for rows.Next() {
		var f models.RepoFile
		var contentHash, language, lastModified, metadataJSON sql.NullString
		var sizeBytes sql.NullInt64

		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Path, &contentHash, &sizeBytes, &language, &lastModified, &f.CreatedAt, &f.UpdatedAt, &metadataJSON); err != nil {
			return nil, 0, fmt.Errorf("failed to scan repo file: %w", err)
		}

		f.ContentHash = models.PtrString(contentHash)
		f.SizeBytes = models.PtrInt64(sizeBytes)
		f.Language = models.PtrString(language)
		f.LastModified = models.PtrString(lastModified)
		if metadataJSON.Valid && metadataJSON.String != "" {
			var meta map[string]interface{}
			if err := models.UnmarshalJSON(metadataJSON.String, &meta); err == nil {
				f.Metadata = meta
			}
		}

		files = append(files, f)
	}

	if files == nil {
		files = []models.RepoFile{}
	}

	return files, total, nil
}

func DeleteRepoFile(db *sql.DB, projectID, path string) error {
	result, err := db.Exec("DELETE FROM repo_files WHERE project_id = ? AND path = ?", projectID, path)
	if err != nil {
		return fmt.Errorf("failed to delete repo file: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("repo file not found: %s/%s", projectID, path)
	}
	return nil
}

func DeleteRepoFilesByProject(db *sql.DB, projectID string) error {
	_, err := db.Exec("DELETE FROM repo_files WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete repo files for project: %w", err)
	}
	return nil
}

func ListRecentRepoFiles(db *sql.DB, projectID string, limit int) ([]models.RepoFile, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, project_id, path, content_hash, size_bytes, language, last_modified, created_at, updated_at, metadata_json
	          FROM repo_files WHERE project_id = ? ORDER BY updated_at DESC LIMIT ?`
	rows, err := db.Query(query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent repo files: %w", err)
	}
	defer rows.Close()

	var files []models.RepoFile
	for rows.Next() {
		var f models.RepoFile
		var contentHash, language, lastModified, metadataJSON sql.NullString
		var sizeBytes sql.NullInt64
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Path, &contentHash, &sizeBytes, &language, &lastModified, &f.CreatedAt, &f.UpdatedAt, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan repo file: %w", err)
		}
		f.ContentHash = models.PtrString(contentHash)
		f.SizeBytes = models.PtrInt64(sizeBytes)
		f.Language = models.PtrString(language)
		f.LastModified = models.PtrString(lastModified)
		if metadataJSON.Valid && metadataJSON.String != "" {
			var meta map[string]interface{}
			if err := models.UnmarshalJSON(metadataJSON.String, &meta); err == nil {
				f.Metadata = meta
			}
		}
		files = append(files, f)
	}
	if files == nil {
		files = []models.RepoFile{}
	}
	return files, nil
}
