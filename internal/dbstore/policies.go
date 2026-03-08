package dbstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

var AllowedPolicyRuleTypes = map[string]struct{}{
	"automation.block":    {},
	"completion.block":    {},
	"task.create.block":   {},
	"task.update.block":   {},
	"task.delete.block":   {},
	"rate_limit":          {},
	"workflow_constraint": {},
	"resource_quota":      {},
	"time_window":         {},
}

func ValidatePolicyRules(rules []models.PolicyRule) error {
	if rules == nil {
		return nil
	}
	for i, rule := range rules {
		if rule == nil {
			return fmt.Errorf("%w: rule %d is null", ErrInvalidPolicyRules, i)
		}
		rawType, ok := rule["type"]
		if !ok {
			return fmt.Errorf("%w: rule %d missing type", ErrInvalidPolicyRules, i)
		}
		ruleType, ok := rawType.(string)
		if !ok {
			return fmt.Errorf("%w: rule %d type must be string", ErrInvalidPolicyRules, i)
		}
		ruleType = strings.ToLower(strings.TrimSpace(ruleType))
		if ruleType == "" {
			return fmt.Errorf("%w: rule %d type is empty", ErrInvalidPolicyRules, i)
		}
		if _, ok := AllowedPolicyRuleTypes[ruleType]; !ok {
			return fmt.Errorf("%w: rule %d has unknown type %s", ErrInvalidPolicyRules, i, ruleType)
		}
		rule["type"] = ruleType
		switch ruleType {
		case "automation.block", "completion.block", "task.create.block", "task.update.block", "task.delete.block":
			if reason, ok := rule["reason"]; ok && reason != nil {
				if _, ok := reason.(string); !ok {
					return fmt.Errorf("%w: rule %d reason must be string", ErrInvalidPolicyRules, i)
				}
			}
		}
	}
	return nil
}

func CreatePolicy(db *sql.DB, projectID, name string, description *string, rules []models.PolicyRule, enabled bool) (*models.Policy, error) {
	if rules == nil {
		rules = []models.PolicyRule{}
	}
	if err := ValidatePolicyRules(rules); err != nil {
		return nil, err
	}

	policy := &models.Policy{
		ID:          "pol_" + uuid.New().String(),
		ProjectID:   projectID,
		Name:        strings.TrimSpace(name),
		Description: description,
		Rules:       rules,
		Enabled:     enabled,
		CreatedAt:   models.NowISO(),
	}

	rulesJSON, err := models.MarshalJSON(policy.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rules: %w", err)
	}

	desc := sql.NullString{}
	if description != nil && strings.TrimSpace(*description) != "" {
		desc = sql.NullString{String: strings.TrimSpace(*description), Valid: true}
	}

	enabledInt := 0
	if policy.Enabled {
		enabledInt = 1
	}

	_, err = db.Exec(`
		INSERT INTO policies (id, project_id, name, description, rules_json, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, policy.ID, policy.ProjectID, policy.Name, desc, rulesJSON, enabledInt, policy.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	if err := emitPolicyLifecycleEvent(db, "policy.created", policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func ListPoliciesByProject(db *sql.DB, projectID string, enabledFilter *bool, limit, offset int, sortField, sortDirection string) ([]models.Policy, int, error) {
	filter := strings.Builder{}
	filter.WriteString(" FROM policies WHERE project_id = ?")
	args := []interface{}{projectID}
	if enabledFilter != nil {
		filter.WriteString(" AND enabled = ?")
		if *enabledFilter {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	orderField := strings.ToLower(strings.TrimSpace(sortField))
	if orderField == "" {
		orderField = "created_at"
	}
	orderDirection := strings.ToLower(strings.TrimSpace(sortDirection))
	if orderDirection == "" {
		orderDirection = "asc"
	}
	if orderDirection != "asc" && orderDirection != "desc" {
		return nil, 0, fmt.Errorf("invalid policy sort direction")
	}

	orderBy := ""
	switch orderField {
	case "created_at":
		if orderDirection == "asc" {
			orderBy = "created_at ASC, name ASC, id ASC"
		} else {
			orderBy = "created_at DESC, name ASC, id ASC"
		}
	case "name":
		if orderDirection == "asc" {
			orderBy = "name ASC, created_at ASC, id ASC"
		} else {
			orderBy = "name DESC, created_at ASC, id ASC"
		}
	default:
		return nil, 0, fmt.Errorf("invalid policy sort field")
	}

	countQuery := "SELECT COUNT(*)" + filter.String()
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count policies: %w", err)
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT id, project_id, name, description, rules_json, enabled, created_at`)
	query.WriteString(filter.String())
	query.WriteString(" ORDER BY ")
	query.WriteString(orderBy)
	listArgs := append([]interface{}{}, args...)
	if limit > 0 {
		query.WriteString(" LIMIT ? OFFSET ?")
		listArgs = append(listArgs, limit, offset)
	}

	rows, err := db.Query(query.String(), listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var policy models.Policy
		var description sql.NullString
		var rulesJSON sql.NullString
		var enabledInt int
		if err := rows.Scan(&policy.ID, &policy.ProjectID, &policy.Name, &description, &rulesJSON, &enabledInt, &policy.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan policy: %w", err)
		}

		policy.Description = models.PtrString(description)
		policy.Enabled = enabledInt != 0
		if rulesJSON.Valid && strings.TrimSpace(rulesJSON.String) != "" {
			if err := models.UnmarshalJSON(rulesJSON.String, &policy.Rules); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal rules: %w", err)
			}
		}
		if policy.Rules == nil {
			policy.Rules = []models.PolicyRule{}
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate policies: %w", err)
	}

	return policies, total, nil
}

func GetPolicy(db *sql.DB, projectID, policyID string) (*models.Policy, error) {
	var policy models.Policy
	var description sql.NullString
	var rulesJSON sql.NullString
	var enabledInt int

	err := db.QueryRow(`
		SELECT id, project_id, name, description, rules_json, enabled, created_at
		FROM policies WHERE project_id = ? AND id = ?
	`, projectID, policyID).Scan(&policy.ID, &policy.ProjectID, &policy.Name, &description, &rulesJSON, &enabledInt, &policy.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	policy.Description = models.PtrString(description)
	policy.Enabled = enabledInt != 0
	if rulesJSON.Valid && strings.TrimSpace(rulesJSON.String) != "" {
		if err := models.UnmarshalJSON(rulesJSON.String, &policy.Rules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
		}
	}
	if policy.Rules == nil {
		policy.Rules = []models.PolicyRule{}
	}

	return &policy, nil
}

func UpdatePolicy(db *sql.DB, projectID, policyID string, name *string, description *string, rules []models.PolicyRule, enabled *bool) (*models.Policy, error) {
	policy, err := GetPolicy(db, projectID, policyID)
	if err != nil {
		return nil, err
	}

	if name != nil {
		policy.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if trimmed == "" {
			policy.Description = nil
		} else {
			policy.Description = &trimmed
		}
	}
	if rules != nil {
		if err := ValidatePolicyRules(rules); err != nil {
			return nil, err
		}
		policy.Rules = rules
	}
	if enabled != nil {
		policy.Enabled = *enabled
	}

	rulesJSON, err := models.MarshalJSON(policy.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rules: %w", err)
	}

	desc := sql.NullString{}
	if policy.Description != nil && strings.TrimSpace(*policy.Description) != "" {
		desc = sql.NullString{String: strings.TrimSpace(*policy.Description), Valid: true}
	}

	enabledInt := 0
	if policy.Enabled {
		enabledInt = 1
	}

	_, err = db.Exec(`
		UPDATE policies SET name = ?, description = ?, rules_json = ?, enabled = ?
		WHERE project_id = ? AND id = ?
	`, policy.Name, desc, rulesJSON, enabledInt, projectID, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	if err := emitPolicyLifecycleEvent(db, "policy.updated", policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func DeletePolicy(db *sql.DB, projectID, policyID string) error {
	policy, err := GetPolicy(db, projectID, policyID)
	if err != nil {
		return err
	}

	result, err := db.Exec("DELETE FROM policies WHERE project_id = ? AND id = ?", projectID, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	if err := emitPolicyLifecycleEvent(db, "policy.deleted", policy); err != nil {
		return err
	}
	return nil
}

func emitPolicyLifecycleEvent(db *sql.DB, kind string, policy *models.Policy) error {
	if policy == nil {
		return nil
	}

	payload := map[string]interface{}{
		"policy_id": policy.ID,
		"name":      policy.Name,
		"enabled":   policy.Enabled,
	}
	if policy.Description != nil && strings.TrimSpace(*policy.Description) != "" {
		payload["description"] = strings.TrimSpace(*policy.Description)
	}

	if _, err := CreateEvent(db, policy.ProjectID, kind, "policy", policy.ID, payload); err != nil {
		return fmt.Errorf("failed to emit %s event: %w", kind, err)
	}
	return nil
}
