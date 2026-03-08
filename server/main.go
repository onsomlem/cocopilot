// FastAPI Task Queue Web Server for Agentic LLM Loop - Go Version
package server

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Version is set at build time via -ldflags, falls back to "dev".
var Version = "dev"

//go:embed static
var staticFS embed.FS

var (
	db                 *sql.DB
	policyEngine       *PolicyEngine
	assignmentSvc      *AssignmentService
	finalizationSvc    *FinalizationService
	sseClients         = make([]v1SSEClient, 0)
	sseMutex           sync.RWMutex
	v2EventSubscribers = make([]v2EventSubscriber, 0)
	v2EventMu          sync.RWMutex
	workdir            = "/tmp"
	workdirMu          sync.RWMutex
	serverStartTime    = time.Now().UTC()
)

type v1SSEClient struct {
	ch        chan string
	projectID *string
	eventType string
}

type v2EventSubscriber struct {
	ch        chan Event
	projectID string
	kind      string
}

func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Set recommended SQLite settings
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL")

	// Run migrations to set up the schema
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}

// Main is the server entry point, called from cmd/cocopilot/main.go.
func Main() {
	cfg, err := loadRuntimeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// CLI mode: handle migration commands
	if len(os.Args) > 1 {
		if err := handleCLI(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Server mode: initialize DB and start web server
	err = initDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	policyEngine = NewPolicyEngine(db)
	assignmentSvc = &AssignmentService{DB: db}
	finalizationSvc = &FinalizationService{DB: db}

	// First-run check: ensure a default project exists
	ensureDefaultProject(db)

	initWebhookNotifier()

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	// Start background job to mark stale agents as offline
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := MarkStaleAgentsOffline(db, 10); err != nil {
					log.Printf("Warning: Failed to mark stale agents offline: %v", err)
				}
			}
		}
	}()

	// Start background job to clean up expired leases
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				count, err := DeleteExpiredLeases(db)
				if err != nil {
					log.Printf("Warning: Failed to delete expired leases: %v", err)
				} else if count > 0 {
					log.Printf("Cleaned up %d expired leases", count)
					go broadcastUpdate(v1EventTypeTasks) // Notify clients about potential task availability
				}
			}
		}
	}()

	if cfg.EventsRetentionDays > 0 || cfg.EventsRetentionMax > 0 {
		// Start background job to prune old events
		go func() {
			interval := resolveEventsPruneInterval(cfg)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					start := time.Now()
					deleted, err := PruneEvents(db, cfg.EventsRetentionDays, cfg.EventsRetentionMax)
					duration := time.Since(start)
					if err != nil {
						if isSQLiteBusyError(err) {
							log.Printf("Events prune skipped: sqlite busy (duration=%s)", duration)
							continue
						}
						log.Printf("Warning: Failed to prune events after %s: %v", duration, err)
					} else {
						log.Printf("Events prune completed: deleted=%d duration=%s", deleted, duration)
					}
				}
			}
		}()
	}

	// Start background job to clean up old automation emissions (hourly, 7 days retention)
	go func() {
		const emissionMaxAge = 7 * 24 * 3600 // 7 days in seconds
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				deleted, err := CleanupOldEmissions(db, emissionMaxAge)
				if err != nil {
					log.Printf("Warning: Failed to cleanup old emissions: %v", err)
				} else if deleted > 0 {
					log.Printf("Cleaned up %d old automation emissions", deleted)
				}
			}
		}
	}()

	log.Printf("Starting Agentic Task Queue server on http://%s", cfg.HTTPAddr)
	if strings.HasPrefix(cfg.HTTPAddr, "0.0.0.0") {
		log.Printf("SECURITY WARNING: Server is listening on all interfaces (0.0.0.0). Set COCO_HTTP_ADDR=127.0.0.1:8080 for local-only access.")
	}

	// Auto-open browser unless --no-browser flag or COCO_NO_BROWSER env is set
	if !cfg.NoBrowser {
		go func() {
			// Wait briefly for the server to start
			time.Sleep(500 * time.Millisecond)
			addr := cfg.HTTPAddr
			if strings.HasPrefix(addr, "0.0.0.0") {
				addr = "127.0.0.1" + addr[len("0.0.0.0"):]
			}
			url := "http://" + addr
			log.Printf("Opening browser at %s", url)
			openBrowser(url)
		}()
	}

	// Periodic stall and idle detection (every 10 minutes).
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			DetectStalledTasks(db, 30*time.Minute)
			DetectIdleProjects(db, 1*time.Hour)
		}
	}()

	handler := withCORS(withRequestLog(mux))
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, handler))
}
