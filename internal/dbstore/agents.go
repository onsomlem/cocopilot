package dbstore

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func RegisterAgent(db *sql.DB, name string, capabilities []string, metadata map[string]interface{}) (*models.Agent, error) {
	agent := &models.Agent{
		ID:           "agent_" + uuid.New().String(),
		Name:         name,
		Capabilities: capabilities,
		Metadata:     metadata,
		Status:       models.AgentStatusOnline,
		LastSeen:     nil,
		RegisteredAt: models.NowISO(),
	}

	capabilitiesJSON, err := models.MarshalJSON(capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	metadataJSON, err := models.MarshalJSON(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO agents (id, name, capabilities_json, metadata_json, status, registered_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, agent.ID, agent.Name, capabilitiesJSON, metadataJSON, agent.Status, agent.RegisteredAt)

	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}
	return agent, nil
}

func GetAgent(db *sql.DB, agentID string) (*models.Agent, error) {
	var agent models.Agent
	var capabilitiesJSON, metadataJSON sql.NullString
	var lastSeen sql.NullString

	err := db.QueryRow(`
		SELECT id, name, capabilities_json, metadata_json, status, last_seen, registered_at
		FROM agents WHERE id = ?
	`, agentID).Scan(&agent.ID, &agent.Name, &capabilitiesJSON, &metadataJSON, &agent.Status, &lastSeen, &agent.RegisteredAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if capabilitiesJSON.Valid && capabilitiesJSON.String != "" {
		if err := models.UnmarshalJSON(capabilitiesJSON.String, &agent.Capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := models.UnmarshalJSON(metadataJSON.String, &agent.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	agent.LastSeen = models.PtrString(lastSeen)

	return &agent, nil
}

func ListAgents(db *sql.DB, statusFilter string, since string, limit int, offset int, sortField string, sortDirection string) ([]models.Agent, int, error) {
	baseQuery := "FROM agents"
	clauses := []string{}
	args := []interface{}{}

	switch statusFilter {
	case "":
	case "active":
		clauses = append(clauses, "status IN (?, ?, ?)")
		args = append(args, models.AgentStatusOnline, models.AgentStatusIdle, models.AgentStatusBusy)
	case "stale":
		clauses = append(clauses, "status = ?")
		args = append(args, models.AgentStatusOffline)
	default:
		return nil, 0, fmt.Errorf("invalid status filter: %s", statusFilter)
	}

	if since != "" {
		clauses = append(clauses, "COALESCE(last_seen, registered_at) >= ?")
		args = append(args, since)
	}

	if len(clauses) > 0 {
		baseQuery += " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(1) " + baseQuery
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	orderBy := "registered_at ASC"
	switch sortField {
	case "", "created_at":
		orderBy = "registered_at ASC"
	case "last_seen":
		if strings.EqualFold(sortDirection, "desc") {
			orderBy = "COALESCE(last_seen, registered_at) DESC"
		} else {
			orderBy = "COALESCE(last_seen, registered_at) ASC"
		}
	default:
		return nil, 0, fmt.Errorf("invalid sort field: %s", sortField)
	}

	query := "SELECT id, name, capabilities_json, metadata_json, status, last_seen, registered_at " + baseQuery + " ORDER BY " + orderBy + " LIMIT ? OFFSET ?"
	argsWithPaging := append(args, limit, offset)
	rows, err := db.Query(query, argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agent models.Agent
		var capabilitiesJSON, metadataJSON sql.NullString
		var lastSeen sql.NullString

		err := rows.Scan(&agent.ID, &agent.Name, &capabilitiesJSON, &metadataJSON, &agent.Status, &lastSeen, &agent.RegisteredAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan agent: %w", err)
		}

		if capabilitiesJSON.Valid && capabilitiesJSON.String != "" {
			if err := models.UnmarshalJSON(capabilitiesJSON.String, &agent.Capabilities); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := models.UnmarshalJSON(metadataJSON.String, &agent.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		agent.LastSeen = models.PtrString(lastSeen)
		agents = append(agents, agent)
	}
	return agents, total, nil
}

func DeleteAgent(db *sql.DB, agentID string) (*models.Agent, error) {
	agent, err := GetAgent(db, agentID)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("DELETE FROM agents WHERE id = ?", agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent: %w", err)
	}

	return agent, nil
}

func UpdateAgentStatus(db *sql.DB, agentID string, status models.AgentStatus) error {
	_, err := GetAgent(db, agentID)
	if err != nil {
		return err
	}

	lastSeen := models.NowISO()
	_, err = db.Exec(`
		UPDATE agents SET status = ?, last_seen = ? WHERE id = ?
	`, status, lastSeen, agentID)

	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}
	return nil
}

func UpdateAgentHeartbeat(db *sql.DB, agentID string) error {
	return UpdateAgentStatus(db, agentID, models.AgentStatusOnline)
}

func MarkStaleAgentsOffline(db *sql.DB, staleThresholdMinutes int) error {
	cutoffTime := time.Now().UTC().Add(-time.Duration(staleThresholdMinutes) * time.Minute)
	cutoffISO := cutoffTime.Format("2006-01-02T15:04:05.000Z")

	_, err := db.Exec(`
		UPDATE agents 
		SET status = ? 
		WHERE status = ? 
		AND (last_seen IS NULL OR last_seen < ?)
	`, models.AgentStatusOffline, models.AgentStatusOnline, cutoffISO)

	if err != nil {
		return fmt.Errorf("failed to mark stale agents offline: %w", err)
	}
	return nil
}

func UpdateAgentCapabilities(db *sql.DB, agentID string, capabilities []string) (*models.Agent, error) {
	capJSON, err := models.MarshalJSON(capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}
	_, err = db.Exec(`UPDATE agents SET capabilities_json = ? WHERE id = ?`, capJSON, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent capabilities: %w", err)
	}
	return GetAgent(db, agentID)
}
