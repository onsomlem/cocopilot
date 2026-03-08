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

// RunWorker implements a simple reference worker that polls claim-next,
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

	for {
		taskID, title, instructions, err := workerClaimNext(client, baseURL, projectID, apiKey)
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

		if taskID == 0 {
			fmt.Printf("[%s] No tasks available, waiting...\n", time.Now().Format("15:04:05"))
			time.Sleep(pollInterval)
			continue
		}

		fmt.Printf("[%s] Claimed task #%d: %s\n", time.Now().Format("15:04:05"), taskID, title)
		if instructions != "" {
			fmt.Printf("  Instructions: %s\n", truncate(instructions, 200))
		}

		result, execErr := executor.Execute(taskID, title, instructions)
		if execErr != nil {
			log.Printf("Worker: executor failed for task #%d: %v", taskID, execErr)
			result = fmt.Sprintf("Worker executor failed: %v", execErr)
		}
		if err := workerCompleteTask(client, baseURL, taskID, apiKey, result); err != nil {
			log.Printf("Worker: failed to complete task #%d: %v", taskID, err)
		} else {
			fmt.Printf("[%s] Completed task #%d\n", time.Now().Format("15:04:05"), taskID)
		}

		time.Sleep(1 * time.Second)
	}
}

func workerClaimNext(client *http.Client, baseURL, projectID, apiKey string) (int, string, string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/tasks/claim-next", baseURL, projectID)

	body := strings.NewReader(`{"agent_id":"agent_worker"}`)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return 0, "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return 0, "", "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	envelope, ok := result["envelope"].(map[string]interface{})
	if !ok {
		if tid, ok := result["task_id"]; ok {
			taskID := toInt(tid)
			return taskID, "", "", nil
		}
		return 0, "", "", fmt.Errorf("unexpected response format")
	}

	taskID := toInt(envelope["task_id"])
	title, _ := envelope["title"].(string)
	instructions, _ := envelope["instructions"].(string)

	return taskID, title, instructions, nil
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
