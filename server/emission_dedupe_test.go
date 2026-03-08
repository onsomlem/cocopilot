package server

import (
	"testing"
)

func TestEmissionDedupeFirstRecordSucceeds(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit1", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	SetEmissionWindow(300) // 5 minutes

	recorded, err := TryRecordEmission(db, "proj_emit1", "idle_planner", nil)
	if err != nil {
		t.Fatalf("TryRecordEmission failed: %v", err)
	}
	if !recorded {
		t.Error("expected first emission to be recorded")
	}
}

func TestEmissionDedupeDuplicateBlocked(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit2", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	SetEmissionWindow(300)

	// First emission should succeed
	recorded1, err := TryRecordEmission(db, "proj_emit2", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded1 {
		t.Error("expected first emission to be recorded")
	}

	// Second emission in same window should be blocked
	recorded2, err := TryRecordEmission(db, "proj_emit2", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if recorded2 {
		t.Error("expected second emission in same window to be blocked")
	}
}

func TestEmissionDedupeDifferentKindsAllowed(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit3", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	SetEmissionWindow(300)

	recorded1, err := TryRecordEmission(db, "proj_emit3", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded1 {
		t.Error("expected first kind emission to be recorded")
	}

	// Different kind should succeed
	recorded2, err := TryRecordEmission(db, "proj_emit3", "review_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded2 {
		t.Error("expected different kind emission to be recorded")
	}
}

func TestEmissionDedupeDifferentProjectsAllowed(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	for _, id := range []string{"proj_emit4a", "proj_emit4b"} {
		_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
			id, "Test", "/tmp", nowISO())
		if err != nil {
			t.Fatal(err)
		}
	}

	SetEmissionWindow(300)

	recorded1, err := TryRecordEmission(db, "proj_emit4a", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded1 {
		t.Error("expected first project emission to be recorded")
	}

	// Different project, same kind should succeed
	recorded2, err := TryRecordEmission(db, "proj_emit4b", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded2 {
		t.Error("expected different project emission to be recorded")
	}
}

func TestCheckEmissionAllowed(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit5", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	SetEmissionWindow(300)

	// Before recording, should be allowed
	allowed, err := CheckEmissionAllowed(db, "proj_emit5", "idle_planner")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected emission to be allowed before recording")
	}

	// Record it
	_, err = TryRecordEmission(db, "proj_emit5", "idle_planner", nil)
	if err != nil {
		t.Fatal(err)
	}

	// After recording, should not be allowed
	allowed, err = CheckEmissionAllowed(db, "proj_emit5", "idle_planner")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("expected emission to NOT be allowed after recording")
	}
}

func TestEmissionWithTaskID(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit6", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	SetEmissionWindow(300)

	taskID := 42
	recorded, err := TryRecordEmission(db, "proj_emit6", "idle_planner", &taskID)
	if err != nil {
		t.Fatalf("TryRecordEmission with taskID failed: %v", err)
	}
	if !recorded {
		t.Error("expected emission with taskID to be recorded")
	}

	// Same window, same kind, different task ID should still be blocked (dedupe is by project+kind+window)
	taskID2 := 99
	recorded2, err := TryRecordEmission(db, "proj_emit6", "idle_planner", &taskID2)
	if err != nil {
		t.Fatal(err)
	}
	if recorded2 {
		t.Error("expected duplicate emission to be blocked even with different taskID")
	}
}

func TestCleanupOldEmissions(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_emit7", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	// Insert an old emission directly
	_, err = db.Exec(`INSERT INTO automation_emissions (dedupe_key, project_id, kind, created_at) VALUES (?, ?, ?, ?)`,
		"old_key_1", "proj_emit7", "idle_planner", 1000) // very old timestamp
	if err != nil {
		t.Fatal(err)
	}

	// Insert a recent emission
	SetEmissionWindow(300)
	_, err = TryRecordEmission(db, "proj_emit7", "review_planner", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup emissions older than 1 hour
	deleted, err := CleanupOldEmissions(db, 3600)
	if err != nil {
		t.Fatalf("CleanupOldEmissions failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 old emission deleted, got %d", deleted)
	}

	// Verify the recent one still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM automation_emissions WHERE project_id = ?", "proj_emit7").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 remaining emission, got %d", count)
	}
}

func TestSetEmissionWindow(t *testing.T) {
	// Set a custom window
	SetEmissionWindow(60)
	if got := GetEmissionWindow(); got != 60 {
		t.Errorf("expected emission window 60, got %d", got)
	}

	// Setting to 0 should use default
	SetEmissionWindow(0)
	if got := GetEmissionWindow(); got <= 0 {
		t.Errorf("expected positive default window, got %d", got)
	}

	// Setting negative should use default
	SetEmissionWindow(-1)
	if got := GetEmissionWindow(); got <= 0 {
		t.Errorf("expected positive default window, got %d", got)
	}
}
