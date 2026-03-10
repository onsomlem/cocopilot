package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/onsomlem/cocopilot/internal/models"
)

func TestLoopAnchorPrompt_CreateAndClaim(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(testDB, "anchor-test", "", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	anchor := "Stay focused on the build task. Do not drift."
	body := `{"instructions":"Build the project","title":"Build","loop_anchor_prompt":"` + anchor + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+proj.ID+"/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Task struct {
			ID               int     `json:"id"`
			LoopAnchorPrompt *string `json:"loop_anchor_prompt"`
		} `json:"task"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if createResp.Task.LoopAnchorPrompt == nil || *createResp.Task.LoopAnchorPrompt != anchor {
		t.Fatalf("expected loop_anchor_prompt=%q, got %v", anchor, createResp.Task.LoopAnchorPrompt)
	}

	// Claim the task and verify loop_anchor_prompt is included
	claimBody := `{"agent_id":"test-agent"}`
	claimReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", createResp.Task.ID), strings.NewReader(claimBody))
	claimReq.Header.Set("Content-Type", "application/json")
	claimRR := httptest.NewRecorder()
	v2TaskClaimHandler(claimRR, claimReq, strconv.Itoa(createResp.Task.ID))

	if claimRR.Code != http.StatusOK {
		t.Fatalf("claim: expected 200, got %d: %s", claimRR.Code, claimRR.Body.String())
	}

	var claimResp struct {
		Task struct {
			LoopAnchorPrompt *string `json:"loop_anchor_prompt"`
		} `json:"task"`
	}
	if err := json.Unmarshal(claimRR.Body.Bytes(), &claimResp); err != nil {
		t.Fatalf("unmarshal claim response: %v", err)
	}
	if claimResp.Task.LoopAnchorPrompt == nil || *claimResp.Task.LoopAnchorPrompt != anchor {
		t.Fatalf("claim: expected loop_anchor_prompt=%q, got %v", anchor, claimResp.Task.LoopAnchorPrompt)
	}
}

func TestLoopAnchorPrompt_DefaultConstant(t *testing.T) {
	if models.DefaultLoopAnchorPrompt == "" {
		t.Fatal("DefaultLoopAnchorPrompt should not be empty")
	}
	if !strings.Contains(models.DefaultLoopAnchorPrompt, "Stay focused") {
		t.Fatalf("DefaultLoopAnchorPrompt should contain 'Stay focused', got: %s", models.DefaultLoopAnchorPrompt)
	}
}

func TestLoopAnchorPrompt_TaskWithoutAnchor(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, _ := CreateProject(db, "no-anchor", "", nil)

	body := `{"instructions":"Simple task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+proj.ID+"/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Task struct {
			LoopAnchorPrompt *string `json:"loop_anchor_prompt"`
		} `json:"task"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Task.LoopAnchorPrompt != nil {
		t.Fatalf("expected nil loop_anchor_prompt for task created without one, got %q", *resp.Task.LoopAnchorPrompt)
	}
}

func TestLoopAnchorPrompt_TemplateInheritance(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, _ := CreateProject(db, "tmpl-anchor", "", nil)
	anchor := "Template-level anchor prompt"
	tmpl, err := CreateTaskTemplate(db, proj.ID, "Anchored", nil, "template task", nil, 50, nil, nil, &anchor)
	if err != nil {
		t.Fatalf("CreateTaskTemplate: %v", err)
	}

	// Instantiate from template
	instURL := "/api/v2/projects/" + proj.ID + "/templates/" + tmpl.ID + "/instantiate"
	req := httptest.NewRequest(http.MethodPost, instURL, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Task struct {
			LoopAnchorPrompt *string `json:"loop_anchor_prompt"`
		} `json:"task"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Task.LoopAnchorPrompt == nil || *resp.Task.LoopAnchorPrompt != anchor {
		t.Fatalf("expected loop_anchor_prompt=%q from template, got %v", anchor, resp.Task.LoopAnchorPrompt)
	}
}
