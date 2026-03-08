package dbstore

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/onsomlem/cocopilot/internal/models"
)

func GetProjectDashboardData(db *sql.DB, projectID string) (*models.DashboardData, error) {
	d := &models.DashboardData{ProjectID: projectID}

	// Task counts by status bucket.
	db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status_v2 = 'QUEUED'`, projectID).Scan(&d.TaskCounts.Queued)
	db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status_v2 IN ('CLAIMED','RUNNING')`, projectID).Scan(&d.TaskCounts.InProgress)
	db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status_v2 = 'SUCCEEDED'`, projectID).Scan(&d.TaskCounts.Completed)
	db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status_v2 = 'FAILED'`, projectID).Scan(&d.TaskCounts.Failed)

	// Last 5 active runs with task title.
	runRows, err := db.Query(`
		SELECT r.id, COALESCE(t.title, ''), r.status, r.started_at
		FROM runs r
		JOIN tasks t ON r.task_id = t.id
		WHERE t.project_id = ? AND r.status = 'RUNNING'
		ORDER BY r.started_at DESC LIMIT 5
	`, projectID)
	if err == nil {
		defer runRows.Close()
		for runRows.Next() {
			var run models.DashboardRun
			if scanErr := runRows.Scan(&run.RunID, &run.TaskTitle, &run.Status, &run.StartedAt); scanErr == nil {
				d.ActiveRuns = append(d.ActiveRuns, run)
			}
		}
	}
	if d.ActiveRuns == nil {
		d.ActiveRuns = []models.DashboardRun{}
	}

	// Last 5 recent repo file changes.
	repoFiles, _ := ListRecentRepoFiles(db, projectID, 5)
	for _, f := range repoFiles {
		d.RecentChanges = append(d.RecentChanges, models.DashboardRepoChange{
			Path:      f.Path,
			UpdatedAt: f.UpdatedAt,
		})
	}
	if d.RecentChanges == nil {
		d.RecentChanges = []models.DashboardRepoChange{}
	}

	// Last 5 failed tasks.
	failRows, err := db.Query(`
		SELECT t.id, COALESCE(t.title,''), COALESCE(r.error,''), t.updated_at
		FROM tasks t
		LEFT JOIN runs r ON r.task_id = t.id AND r.status = 'FAILED'
		WHERE t.project_id = ? AND t.status_v2 = 'FAILED'
		ORDER BY t.updated_at DESC LIMIT 5
	`, projectID)
	if err == nil {
		defer failRows.Close()
		for failRows.Next() {
			var f models.DashboardFailure
			if scanErr := failRows.Scan(&f.TaskID, &f.Title, &f.Error, &f.UpdatedAt); scanErr == nil {
				d.RecentFailures = append(d.RecentFailures, f)
			}
		}
	}
	if d.RecentFailures == nil {
		d.RecentFailures = []models.DashboardFailure{}
	}

	// Last 5 automation events.
	evRows, err := db.Query(`
		SELECT kind, created_at FROM events
		WHERE project_id = ? AND kind LIKE 'automation.%'
		ORDER BY created_at DESC LIMIT 5
	`, projectID)
	if err == nil {
		defer evRows.Close()
		for evRows.Next() {
			var ae models.DashboardAutoEvent
			if scanErr := evRows.Scan(&ae.Kind, &ae.CreatedAt); scanErr == nil {
				d.AutomationEvents = append(d.AutomationEvents, ae)
			}
		}
	}
	if d.AutomationEvents == nil {
		d.AutomationEvents = []models.DashboardAutoEvent{}
	}

	// Recommendations: up to 3 queued tasks, scored by multiple signals.
	type recCandidate struct {
		taskID   int
		title    string
		priority int
		taskType string
		signal   int
		reasons  []string
	}
	candidates := map[int]*recCandidate{}

	qRows, qErr := db.Query(`
		SELECT id, COALESCE(title,''), priority, COALESCE(type,'')
		FROM tasks WHERE project_id = ? AND status_v2 = ?
		ORDER BY priority DESC, id ASC LIMIT 50
	`, projectID, string(models.TaskStatusQueued))
	if qErr == nil {
		defer qRows.Close()
		for qRows.Next() {
			var c recCandidate
			if scanErr := qRows.Scan(&c.taskID, &c.title, &c.priority, &c.taskType); scanErr == nil {
				c.signal = 0
				if c.priority >= 3 {
					c.signal += 2
					c.reasons = append(c.reasons, fmt.Sprintf("High priority (P%d)", c.priority))
				} else if c.priority > 0 {
					c.signal++
					c.reasons = append(c.reasons, fmt.Sprintf("Priority P%d", c.priority))
				}
				if c.taskType != "" {
					c.reasons = append(c.reasons, fmt.Sprintf("Type: %s", c.taskType))
				}
				candidates[c.taskID] = &c
			}
		}
	}

	// Signal: tasks whose dependencies were recently completed.
	depRows, depErr := db.Query(`
		SELECT td.task_id, dep.id, COALESCE(dep.title,'')
		FROM task_dependencies td
		JOIN tasks dep ON dep.id = td.depends_on_task_id
		WHERE dep.project_id = ? AND dep.status_v2 = 'SUCCEEDED'
		  AND dep.updated_at >= datetime('now', '-1 hour')
	`, projectID)
	if depErr == nil {
		defer depRows.Close()
		for depRows.Next() {
			var tid, depID int
			var depTitle string
			if depRows.Scan(&tid, &depID, &depTitle) == nil {
				if c, ok := candidates[tid]; ok {
					c.signal += 3
					evidence := fmt.Sprintf("Dependency 'task-%d' just completed", depID)
					if depTitle != "" {
						evidence = fmt.Sprintf("Dependency '%s' (task-%d) just completed", depTitle, depID)
					}
					c.reasons = append(c.reasons, evidence)
				}
			}
		}
	}

	// Signal: tasks whose title matches recently-changed repo files.
	if len(d.RecentChanges) > 0 {
		for _, c := range candidates {
			lowerTitle := strings.ToLower(c.title)
			for _, rc := range d.RecentChanges {
				base := rc.Path
				if idx := strings.LastIndex(base, "/"); idx >= 0 {
					base = base[idx+1:]
				}
				if idx := strings.LastIndex(base, "."); idx >= 0 {
					base = base[:idx]
				}
				if len(base) > 3 && strings.Contains(lowerTitle, strings.ToLower(base)) {
					c.signal += 2
					c.reasons = append(c.reasons, fmt.Sprintf("Related repo change in %s", rc.Path))
					break
				}
			}
		}
	}

	// Signal: tasks with failed siblings (same parent) — may need attention.
	for _, f := range d.RecentFailures {
		for _, c := range candidates {
			if c.taskType == "TEST" || c.taskType == "REVIEW" {
				lowerTitle := strings.ToLower(c.title)
				failTitle := strings.ToLower(f.Title)
				for _, w := range strings.Fields(failTitle) {
					w = strings.Trim(w, ".,;:!?()[]{}\"'")
					if len(w) > 4 && strings.Contains(lowerTitle, w) {
						c.signal++
						c.reasons = append(c.reasons, fmt.Sprintf("Related task-%d recently failed", f.TaskID))
						break
					}
				}
			}
		}
	}

	// Sort candidates by signal desc, then id asc; take top 3.
	type scored struct {
		c *recCandidate
	}
	sortedCandidates := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		sortedCandidates = append(sortedCandidates, scored{c})
	}
	sort.Slice(sortedCandidates, func(i, j int) bool {
		if sortedCandidates[i].c.signal != sortedCandidates[j].c.signal {
			return sortedCandidates[i].c.signal > sortedCandidates[j].c.signal
		}
		return sortedCandidates[i].c.taskID < sortedCandidates[j].c.taskID
	})
	limit := 3
	if len(sortedCandidates) < limit {
		limit = len(sortedCandidates)
	}
	d.Recommendations = make([]models.DashboardRecommendation, 0, limit)
	for _, sc := range sortedCandidates[:limit] {
		why := "Queued"
		if len(sc.c.reasons) > 0 {
			why = strings.Join(sc.c.reasons, "; ")
		}
		d.Recommendations = append(d.Recommendations, models.DashboardRecommendation{
			TaskID:   sc.c.taskID,
			Title:    sc.c.title,
			Priority: sc.c.priority,
			Type:     sc.c.taskType,
			Why:      why,
		})
	}

	// Stalled tasks.
	d.StalledTasks = QueryStalledTasks(db, projectID)

	return d, nil
}

// QueryStalledTasks finds tasks that are CLAIMED or RUNNING but appear stuck:
// either their lease has expired, or they have been in-progress for over 1 hour.
func QueryStalledTasks(db *sql.DB, projectID string) []models.DashboardStalledTask {
	now := models.NowISO()
	rows, err := db.Query(`
		SELECT t.id, COALESCE(t.title,''), t.status_v2,
		       COALESCE(l.agent_id,''), COALESCE(l.created_at,''),
		       CASE
		           WHEN l.expires_at IS NOT NULL AND l.expires_at <= ? THEN 'lease_expired'
		           WHEN t.updated_at <= datetime(?, '-1 hour') THEN 'stalled_over_1h'
		           ELSE 'unknown'
		       END AS reason
		FROM tasks t
		LEFT JOIN leases l ON l.task_id = t.id
		WHERE t.project_id = ?
		  AND t.status_v2 IN ('CLAIMED','RUNNING')
		  AND (
		      (l.expires_at IS NOT NULL AND l.expires_at <= ?)
		      OR t.updated_at <= datetime(?, '-1 hour')
		  )
		ORDER BY t.updated_at ASC
		LIMIT 10
	`, now, now, projectID, now, now)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []models.DashboardStalledTask
	for rows.Next() {
		var st models.DashboardStalledTask
		if err := rows.Scan(&st.TaskID, &st.Title, &st.Status, &st.AgentID, &st.ClaimedAt, &st.Reason); err == nil {
			result = append(result, st)
		}
	}
	if result == nil {
		return []models.DashboardStalledTask{}
	}
	return result
}
