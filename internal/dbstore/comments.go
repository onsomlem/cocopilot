package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateArtifactComment(db *sql.DB, artifactID string, projectID string, lineNumber int, body string, author string) (*models.ArtifactComment, error) {
	id := uuid.New().String()
	now := models.NowISO()
	_, err := db.Exec(
		`INSERT INTO artifact_comments (id, artifact_id, project_id, line_number, body, author, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, artifactID, projectID, lineNumber, body, author, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact comment: %w", err)
	}
	return &models.ArtifactComment{
		ID: id, ArtifactID: artifactID, ProjectID: projectID,
		LineNumber: lineNumber, Body: body, Author: author,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func ListArtifactComments(db *sql.DB, artifactID string) ([]models.ArtifactComment, error) {
	rows, err := db.Query(
		`SELECT id, artifact_id, project_id, line_number, body, author, created_at, updated_at FROM artifact_comments WHERE artifact_id = ? ORDER BY line_number ASC, created_at ASC`,
		artifactID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact comments: %w", err)
	}
	defer rows.Close()
	var comments []models.ArtifactComment
	for rows.Next() {
		var c models.ArtifactComment
		if err := rows.Scan(&c.ID, &c.ArtifactID, &c.ProjectID, &c.LineNumber, &c.Body, &c.Author, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan artifact comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func DeleteArtifactComment(db *sql.DB, commentID string) error {
	result, err := db.Exec("DELETE FROM artifact_comments WHERE id = ?", commentID)
	if err != nil {
		return fmt.Errorf("failed to delete artifact comment: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}
