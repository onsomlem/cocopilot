package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2Templates_CreateAndList(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, err := CreateProject(db, "tmpl-test", "", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	baseURL := "/api/v2/projects/" + proj.ID + "/templates"

	// POST — create a template
	body := `{"name":"Bug Fix","description":"Fix a bug","instructions":"Fix it","default_type":"bug","default_priority":80,"default_tags":["urgent"]}`
	req := httptest.NewRequest(http.MethodPost, baseURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectTemplatesHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create template returned %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	tmpl := createResp["template"].(map[string]interface{})
	templateID := tmpl["id"].(string)

	if tmpl["name"].(string) != "Bug Fix" {
		t.Errorf("expected name 'Bug Fix', got %s", tmpl["name"])
	}
	if int(tmpl["default_priority"].(float64)) != 80 {
		t.Errorf("expected priority 80, got %v", tmpl["default_priority"])
	}

	// GET — list templates
	req2 := httptest.NewRequest(http.MethodGet, baseURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectTemplatesHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list templates returned %d", w2.Code)
	}

	var listResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &listResp)
	templates := listResp["templates"].([]interface{})
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}

	t.Logf("created template: %s", templateID)
}

func TestV2Templates_GetUpdateDelete(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "tmpl-test2", "", nil)
	tmpl, err := CreateTaskTemplate(db, proj.ID, "Original", nil, "do stuff", nil, 50, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateTaskTemplate: %v", err)
	}

	detailURL := "/api/v2/projects/" + proj.ID + "/templates/" + tmpl.ID

	// GET
	req := httptest.NewRequest(http.MethodGet, detailURL, nil)
	w := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get template returned %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp["template"].(map[string]interface{})["name"].(string) != "Original" {
		t.Error("expected template name 'Original'")
	}

	// PUT — update
	body := `{"name":"Updated","instructions":"new instructions"}`
	req2 := httptest.NewRequest(http.MethodPut, detailURL, strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("update template returned %d: %s", w2.Code, w2.Body.String())
	}

	var updateResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &updateResp)
	if updateResp["template"].(map[string]interface{})["name"].(string) != "Updated" {
		t.Error("expected updated name 'Updated'")
	}

	// DELETE
	req3 := httptest.NewRequest(http.MethodDelete, detailURL, nil)
	w3 := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("delete template returned %d: %s", w3.Code, w3.Body.String())
	}

	// GET after delete — should 404
	req4 := httptest.NewRequest(http.MethodGet, detailURL, nil)
	w4 := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w4, req4)

	if w4.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w4.Code)
	}
}

func TestV2Templates_Instantiate(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "tmpl-inst", "", nil)
	desc := "Template description"
	tmpl, err := CreateTaskTemplate(db, proj.ID, "Deploy", &desc, "deploy to prod", nil, 90, []string{"deploy"}, nil, nil)
	if err != nil {
		t.Fatalf("CreateTaskTemplate: %v", err)
	}

	instURL := "/api/v2/projects/" + proj.ID + "/templates/" + tmpl.ID + "/instantiate"

	// Instantiate with no overrides
	req := httptest.NewRequest(http.MethodPost, instURL, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("instantiate returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	task := resp["task"].(map[string]interface{})
	if task["instructions"].(string) != "deploy to prod" {
		t.Errorf("expected instructions from template, got %s", task["instructions"])
	}
	if resp["template_id"].(string) != tmpl.ID {
		t.Errorf("expected template_id %s, got %s", tmpl.ID, resp["template_id"])
	}
}

func TestV2Templates_InstantiateWithOverrides(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "tmpl-override", "", nil)
	tmpl, _ := CreateTaskTemplate(db, proj.ID, "Base", nil, "base instructions", nil, 50, []string{"base"}, nil, nil)

	instURL := "/api/v2/projects/" + proj.ID + "/templates/" + tmpl.ID + "/instantiate"
	body := `{"title":"Custom Title","instructions":"custom instructions","priority":99,"tags":["custom"]}`
	req := httptest.NewRequest(http.MethodPost, instURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("instantiate returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	task := resp["task"].(map[string]interface{})

	if task["instructions"].(string) != "custom instructions" {
		t.Errorf("expected overridden instructions")
	}
	if int(task["priority"].(float64)) != 99 {
		t.Errorf("expected priority 99, got %v", task["priority"])
	}
}

func TestV2Templates_CreateMissingName(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "tmpl-noname", "", nil)
	baseURL := "/api/v2/projects/" + proj.ID + "/templates"

	body := `{"instructions":"do stuff"}`
	req := httptest.NewRequest(http.MethodPost, baseURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectTemplatesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestV2Templates_WrongProjectTemplate(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj1, _ := CreateProject(db, "proj1", "", nil)
	proj2, _ := CreateProject(db, "proj2", "", nil)
	tmpl, _ := CreateTaskTemplate(db, proj1.ID, "Proj1Tmpl", nil, "instr", nil, 50, nil, nil, nil)

	// Try to GET template from wrong project
	detailURL := "/api/v2/projects/" + proj2.ID + "/templates/" + tmpl.ID
	req := httptest.NewRequest(http.MethodGet, detailURL, nil)
	w := httptest.NewRecorder()
	v2ProjectTemplateDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for template in wrong project, got %d", w.Code)
	}
}

func TestV2Templates_MethodNotAllowed(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	url := "/api/v2/projects/someproj/templates"
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	v2ProjectTemplatesHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
