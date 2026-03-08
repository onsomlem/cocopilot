package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func handleCLI(cfg runtimeConfig) error {
	command := os.Args[1]

	// Open database for CLI commands
	var err error
	db, err = sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	switch command {
	case "migrate":
		if len(os.Args) > 2 {
			subcommand := os.Args[2]
			switch subcommand {
			case "up":
				return runMigrations(db)
			case "down":
				return rollbackLastMigration(db)
			case "status":
				return getMigrationStatus(db)
			default:
				return fmt.Errorf("unknown migrate subcommand: %s", subcommand)
			}
		}
		return runMigrations(db) // Default to "up"
	case "help", "--help", "-h":
		printHelp()
		return nil
	case "quickstart":
		return handleQuickstart(cfg)
	case "worker":
		projectID := DefaultProjectID
		for _, arg := range os.Args[2:] {
			if strings.HasPrefix(arg, "--project=") {
				projectID = strings.TrimPrefix(arg, "--project=")
			}
		}
		// Init DB and run migrations for worker mode
		if err := initDB(cfg.DBPath); err != nil {
			return fmt.Errorf("failed to initialize database: %v", err)
		}
		return runWorker(projectID)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func handleQuickstart(cfg runtimeConfig) error {
	// Ensure DB + migrations are applied
	if err := initDB(cfg.DBPath); err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	// Health diagnostics
	schemaVer := getCurrentSchemaVersion()
	var taskCount, projectCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
	_ = db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)

	// Ensure default project exists
	project, err := GetProject(db, DefaultProjectID)
	if err != nil || project == nil {
		_, err = CreateProject(db, "default", "/tmp", nil)
		if err != nil {
			return fmt.Errorf("failed to create default project: %v", err)
		}
		projectCount++
	}

	addr := cfg.HTTPAddr
	if strings.HasPrefix(addr, "0.0.0.0") {
		addr = "127.0.0.1" + addr[len("0.0.0.0"):]
	}
	dashURL := "http://" + addr

	fmt.Println("=== Cocopilot Quickstart ===")
	fmt.Println()
	fmt.Println("  Health Diagnostics:")
	fmt.Printf("    DB Status:      OK (%s)\n", cfg.DBPath)
	fmt.Printf("    Schema Version: %d\n", schemaVer)
	fmt.Printf("    Projects:       %d\n", projectCount)
	fmt.Printf("    Tasks:          %d\n", taskCount)
	fmt.Printf("    Listen Address: %s\n", cfg.HTTPAddr)
	fmt.Println()
	fmt.Printf("  Dashboard:  %s\n", dashURL)
	fmt.Printf("  API:        %s/api/v2/health\n", dashURL)
	fmt.Printf("  Project:    %s\n", DefaultProjectID)
	fmt.Println()
	fmt.Println("Starting server...")

	// Close CLI DB handle — server mode will reopen
	db.Close()
	db = nil

	// Now start in server mode
	if err := initDB(cfg.DBPath); err != nil {
		return fmt.Errorf("failed to reinit database: %v", err)
	}

	policyEngine = NewPolicyEngine(db)
	assignmentSvc = &AssignmentService{DB: db}
	finalizationSvc = &FinalizationService{DB: db}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(dashURL)
	}()

	log.Printf("Quickstart server running on %s", dashURL)
	return http.ListenAndServe(cfg.HTTPAddr, mux)
}
