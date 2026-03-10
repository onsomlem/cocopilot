// Package worker implements a reference worker that polls for tasks and executes them.
package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/onsomlem/cocopilot/internal/models"
)

// TaskExecutor defines the interface for pluggable task execution strategies.
type TaskExecutor interface {
	Execute(taskID int, title, instructions string) (result string, err error)
}

// PlaceholderExecutor returns a canned result without doing real work.
type PlaceholderExecutor struct{}

func (e *PlaceholderExecutor) Execute(taskID int, title, _ string) (string, error) {
	return fmt.Sprintf("Worker completed task #%d (%s) at %s (placeholder result)", taskID, title, models.NowISO()), nil
}

// ScriptExecutor runs a shell command with task details as environment variables.
type ScriptExecutor struct{ Command string }

func (e *ScriptExecutor) Execute(taskID int, title, instructions string) (string, error) {
	cmd := exec.Command("sh", "-c", e.Command)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TASK_ID=%d", taskID),
		fmt.Sprintf("TASK_TITLE=%s", title),
		fmt.Sprintf("TASK_INSTRUCTIONS=%s", instructions),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("script failed: %w: %s", err, string(out))
	}
	return string(out), nil
}

// WebhookExecutor sends task details to a URL and returns the response body.
type WebhookExecutor struct {
	URL    string
	Client *http.Client
}

func (e *WebhookExecutor) Execute(taskID int, title, instructions string) (string, error) {
	payload := fmt.Sprintf(`{"task_id":%d,"title":%q,"instructions":%q}`, taskID, title, instructions)
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Post(e.URL, "application/json", strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ResolveExecutor picks a TaskExecutor based on environment configuration.
func ResolveExecutor() TaskExecutor {
	if cmd := os.Getenv("COCO_WORKER_SCRIPT"); cmd != "" {
		return &ScriptExecutor{Command: cmd}
	}
	if url := os.Getenv("COCO_WORKER_WEBHOOK"); url != "" {
		return &WebhookExecutor{URL: url}
	}
	return &PlaceholderExecutor{}
}

// RunWorker implements a reference worker that polls claim-next,
// prints claimed tasks, and marks them as completed with a placeholder result.
func RunWorker(projectID string) error {
	const defaultHTTPAddr = "127.0.0.1:8080"

	httpAddr := os.Getenv("COCO_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}
	if strings.HasPrefix(httpAddr, "0.0.0.0") {
		httpAddr = "127.0.0.1" + httpAddr[len("0.0.0.0"):]
	}
	baseURL := "http://" + httpAddr

	apiKey := strings.TrimSpace(os.Getenv("COCO_API_KEY"))

	client := &http.Client{Timeout: 30 * time.Second}
	pollInterval := 10 * time.Second
	maxConsecutiveFailures := 20
	consecutiveFailures := 0
	maxBackoff := 5 * time.Minute

	executor := ResolveExecutor()
	fmt.Printf("  Executor:  %T\n", executor)

	fmt.Printf("Cocopilot Worker started\n")
	fmt.Printf("  Server:   %s\n", baseURL)
	fmt.Printf("  Project:  %s\n", projectID)
	fmt.Printf("  Polling every %s\n", pollInterval)
	fmt.Printf("  Max consecutive failures: %d\n", maxConsecutiveFailures)
	fmt.Println()

	// Register agent at startup
	if err := workerRegisterAgent(client, baseURL, apiKey); err != nil {
		log.Printf("Worker: agent registration warning: %v (continuing anyway)", err)
	} else {
		fmt.Printf("Worker: agent registered as agent_worker\n")
	}

	// Start heartbeat goroutine
	heartbeatDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				workerRegisterAgent(client, baseURL, apiKey)
			case <-heartbeatDone:
				return
			}
		}
	}()
	defer close(heartbeatDone)

	for {
		claim, err := workerClaimNext(client, baseURL, projectID, apiKey)
		if err != nil {
			consecutiveFailures++
			log.Printf("Worker: claim-next error (%d/%d): %v", consecutiveFailures, maxConsecutiveFailures, err)
			if consecutiveFailures >= maxConsecutiveFailures {
				log.Printf("Worker: %d consecutive failures reached, shutting down", maxConsecutiveFailures)
				return fmt.Errorf("too many consecutive failures (%d)", maxConsecutiveFailures)
			}
			backoff := pollInterval * time.Duration(1<<uint(consecutiveFailures-1))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			log.Printf("Worker: backing off for %s", backoff)
			time.Sleep(backoff)
			continue
		}
		consecutiveFailures = 0

		if claim.TaskID == 0 {
			fmt.Printf("[%s] No tasks available, waiting...\n", time.Now().Format("15:04:05"))
			time.Sleep(pollInterval)
			continue
		}

		fmt.Printf("[%s] Claimed task #%d: %s\n", time.Now().Format("15:04:05"), claim.TaskID, claim.Title)
		if claim.Instructions != "" {
			fmt.Printf("  Instructions: %s\n", truncate(claim.Instructions, 200))
		}

		// Log run step: starting execution
		if claim.RunID != "" {
			workerLogRunStep(client, baseURL, claim.RunID, apiKey, "execute", "running", "Starting task execution")
		}

		// Start lease heartbeat for this task
		leaseStop := make(chan struct{})
		if claim.LeaseID != "" {
			go func(leaseID string) {
				ticker := time.NewTicker(15 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						workerHeartbeatLease(client, baseURL, leaseID, apiKey)
					case <-leaseStop:
						return
					}
				}
			}(claim.LeaseID)
		}

		result, execErr := executor.Execute(claim.TaskID, claim.Title, claim.Instructions)
		close(leaseStop)

		if execErr != nil {
			log.Printf("Worker: executor failed for task #%d: %v", claim.TaskID, execErr)
			// Log failure step
			if claim.RunID != "" {
				workerLogRunStep(client, baseURL, claim.RunID, apiKey, "execute", "failed", fmt.Sprintf("Execution failed: %v", execErr))
			}
			if err := workerFailTask(client, baseURL, claim.TaskID, apiKey, execErr.Error()); err != nil {
				log.Printf("Worker: failed to fail task #%d: %v", claim.TaskID, err)
			} else {
				fmt.Printf("[%s] Failed task #%d\n", time.Now().Format("15:04:05"), claim.TaskID)
			}
		} else {
			// Log success step
			if claim.RunID != "" {
				workerLogRunStep(client, baseURL, claim.RunID, apiKey, "execute", "succeeded", "Task execution completed")
			}
			if err := workerCompleteTask(client, baseURL, claim.TaskID, apiKey, result); err != nil {
				log.Printf("Worker: failed to complete task #%d: %v", claim.TaskID, err)
			} else {
				fmt.Printf("[%s] Completed task #%d\n", time.Now().Format("15:04:05"), claim.TaskID)
			}
		}

		time.Sleep(1 * time.Second)
	}
}

// claimResult holds the parsed claim-next response.
type claimResult struct {
	TaskID       int
	Title        string
	Instructions string
	RunID        string
	LeaseID      string
}

func workerClaimNext(client *http.Client, baseURL, projectID, apiKey string) (claimResult, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/tasks/claim-next", baseURL, projectID)

	body := strings.NewReader(`{"agent_id":"agent_worker"}`)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return claimResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return claimResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return claimResult{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return claimResult{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return claimResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// v2 claim-next returns {"task": {...}, "lease": {...}, "run": {...}, "context": {...}}
	task, ok := result["task"].(map[string]interface{})
	if !ok {
		// Legacy fallback: envelope format
		if envelope, ok := result["envelope"].(map[string]interface{}); ok {
			taskID := toInt(envelope["task_id"])
			title, _ := envelope["title"].(string)
			instructions, _ := envelope["instructions"].(string)
			return claimResult{TaskID: taskID, Title: title, Instructions: instructions}, nil
		}
		if tid, ok := result["task_id"]; ok {
			taskID := toInt(tid)
			return claimResult{TaskID: taskID}, nil
		}
		return claimResult{}, fmt.Errorf("unexpected response format")
	}

	cr := claimResult{
		TaskID:       toInt(task["id"]),
		Title:        toString(task["title"]),
		Instructions: toString(task["instructions"]),
	}

	// Extract run ID
	if run, ok := result["run"].(map[string]interface{}); ok {
		cr.RunID = toString(run["id"])
	}

	// Extract lease ID
	if lease, ok := result["lease"].(map[string]interface{}); ok {
		cr.LeaseID = toString(lease["id"])
	}

	return cr, nil
}

func workerCompleteTask(client *http.Client, baseURL string, taskID int, apiKey, result string) error {
	url := fmt.Sprintf("%s/api/v2/tasks/%d/complete", baseURL, taskID)

	payload := map[string]interface{}{
		"status":  "SUCCEEDED",
		"output":  result,
		"summary": result,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("complete returned status %d", resp.StatusCode)
	}
	return nil
}

func workerFailTask(client *http.Client, baseURL string, taskID int, apiKey, errMsg string) error {
	url := fmt.Sprintf("%s/api/v2/tasks/%d/fail", baseURL, taskID)

	payload := map[string]interface{}{
		"error":  errMsg,
		"output": "Worker execution failed: " + errMsg,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("fail returned status %d", resp.StatusCode)
	}
	return nil
}

func workerRegisterAgent(client *http.Client, baseURL, apiKey string) error {
	url := baseURL + "/api/v2/agents"

	payload := map[string]interface{}{
		"id":   "agent_worker",
		"name": "Built-in Worker",
		"capabilities": []string{"execute", "analyze", "modify"},
		"metadata": map[string]interface{}{
			"type":    "built-in",
			"version": "1.0",
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func workerHeartbeatLease(client *http.Client, baseURL, leaseID, apiKey string) {
	url := fmt.Sprintf("%s/api/v2/leases/%s/heartbeat", baseURL, leaseID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func workerLogRunStep(client *http.Client, baseURL, runID, apiKey, name, status, details string) {
	url := fmt.Sprintf("%s/api/v2/runs/%s/steps", baseURL, runID)

	payload := map[string]interface{}{
		"name":    name,
		"status":  status,
		"details": details,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toInt(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
