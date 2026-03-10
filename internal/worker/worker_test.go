package worker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPlaceholderExecutor(t *testing.T) {
	e := &PlaceholderExecutor{}
	result, err := e.Execute(42, "Test Task", "do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestScriptExecutor(t *testing.T) {
	e := &ScriptExecutor{Command: "echo hello"}
	result, err := e.Execute(1, "test", "inst")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", result)
	}
}

func TestScriptExecutorFailure(t *testing.T) {
	e := &ScriptExecutor{Command: "exit 1"}
	_, err := e.Execute(1, "test", "inst")
	if err == nil {
		t.Fatal("expected error for failing script")
	}
}

func TestScriptExecutorEnvVars(t *testing.T) {
	e := &ScriptExecutor{Command: "echo $TASK_ID $TASK_TITLE"}
	result, err := e.Execute(99, "MyTitle", "instructions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "99 MyTitle\n" {
		t.Fatalf("expected '99 MyTitle\\n', got %q", result)
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", "hello"},
		{42, ""},
		{nil, ""},
		{true, ""},
	}
	for _, tc := range tests {
		got := toString(tc.input)
		if got != tc.expected {
			t.Errorf("toString(%v) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected int
	}{
		{float64(42), 42},
		{int(7), 7},
		{int64(99), 99},
		{"123", 123},
		{"abc", 0},
		{nil, 0},
		{true, 0},
	}
	for _, tc := range tests {
		got := toInt(tc.input)
		if got != tc.expected {
			t.Errorf("toInt(%v) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Error("should not truncate short strings")
	}
	if truncate("long string here", 4) != "long..." {
		t.Errorf("truncate('long string here', 4) = %q", truncate("long string here", 4))
	}
	if truncate("", 5) != "" {
		t.Error("empty string should stay empty")
	}
}

func TestResolveExecutor(t *testing.T) {
	e := ResolveExecutor()
	if _, ok := e.(*PlaceholderExecutor); !ok {
		t.Fatalf("expected PlaceholderExecutor, got %T", e)
	}
}

func TestWebhookExecutor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)
		if payload["task_id"] != float64(5) {
			t.Errorf("expected task_id=5, got %v", payload["task_id"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook result"))
	}))
	defer server.Close()

	e := &WebhookExecutor{URL: server.URL, Client: server.Client()}
	result, err := e.Execute(5, "Test", "inst")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "webhook result" {
		t.Fatalf("expected 'webhook result', got %q", result)
	}
}

func TestWebhookExecutorError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e := &WebhookExecutor{URL: server.URL, Client: server.Client()}
	_, err := e.Execute(1, "Test", "inst")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestWorkerRegistrationContract(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/agents" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &received)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"agent_worker"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := workerRegisterAgent(server.Client(), server.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["id"] != "agent_worker" {
		t.Errorf("expected agent id 'agent_worker', got %v", received["id"])
	}
	if received["name"] != "Built-in Worker" {
		t.Errorf("expected agent name 'Built-in Worker', got %v", received["name"])
	}
}

func TestWorkerRunStepContract(t *testing.T) {
	var mu sync.Mutex
	var steps []map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var step map[string]interface{}
			json.Unmarshal(body, &step)
			mu.Lock()
			steps = append(steps, step)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"step_1"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	workerLogRunStep(server.Client(), server.URL, "run_1", "", "execute", "STARTED", "Starting")
	workerLogRunStep(server.Client(), server.URL, "run_1", "", "execute", "SUCCEEDED", "Done")
	workerLogRunStep(server.Client(), server.URL, "run_1", "", "execute", "FAILED", "Error")

	mu.Lock()
	defer mu.Unlock()

	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	expectedStatuses := []string{"STARTED", "SUCCEEDED", "FAILED"}
	for i, expected := range expectedStatuses {
		if steps[i]["status"] != expected {
			t.Errorf("step %d: expected status %q, got %q", i, expected, steps[i]["status"])
		}
	}

	for i, step := range steps {
		details, ok := step["details"].(map[string]interface{})
		if !ok {
			t.Errorf("step %d: expected details to be a map, got %T", i, step["details"])
			continue
		}
		if _, ok := details["message"]; !ok {
			t.Errorf("step %d: expected details.message field", i)
		}
	}
}

func TestWorkerCompleteContract(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := workerCompleteTask(server.Client(), server.URL, 42, "", "task result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["status"] != "SUCCEEDED" {
		t.Errorf("expected status SUCCEEDED, got %v", received["status"])
	}
	if received["output"] != "task result" {
		t.Errorf("expected output 'task result', got %v", received["output"])
	}
}

func TestWorkerFailContract(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := workerFailTask(server.Client(), server.URL, 42, "", "something broke")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["error"] != "something broke" {
		t.Errorf("expected error 'something broke', got %v", received["error"])
	}
}

func TestWorkerClaimNextParsesV2Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"task": map[string]interface{}{
				"id":           float64(7),
				"title":        "Test Task",
				"instructions": "Do this",
			},
			"lease": map[string]interface{}{
				"id": "lease_abc",
			},
			"run": map[string]interface{}{
				"id": "run_xyz",
			},
		})
	}))
	defer server.Close()

	claim, err := workerClaimNext(server.Client(), server.URL, "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claim.TaskID != 7 {
		t.Errorf("expected TaskID=7, got %d", claim.TaskID)
	}
	if claim.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", claim.Title)
	}
	if claim.Instructions != "Do this" {
		t.Errorf("expected instructions 'Do this', got %q", claim.Instructions)
	}
	if claim.LeaseID != "lease_abc" {
		t.Errorf("expected LeaseID 'lease_abc', got %q", claim.LeaseID)
	}
	if claim.RunID != "run_xyz" {
		t.Errorf("expected RunID 'run_xyz', got %q", claim.RunID)
	}
}

func TestWorkerClaimNextNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	claim, err := workerClaimNext(server.Client(), server.URL, "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claim.TaskID != 0 {
		t.Errorf("expected TaskID=0 for no content, got %d", claim.TaskID)
	}
}

func TestWorkerLeaseHeartbeat(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/leases/lease_123/heartbeat" && r.Method == http.MethodPost {
			called = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := workerHeartbeatLease(server.Client(), server.URL, "lease_123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("lease heartbeat was not called")
	}
}
