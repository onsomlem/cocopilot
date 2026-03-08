package server

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

func printHelp() {
	fmt.Println("Cocopilot Task Queue Server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cocopilot                    Start the web server")
	fmt.Println("  cocopilot quickstart         Create default project, open browser, start server")
	fmt.Println("  cocopilot worker             Start a reference worker that polls and completes tasks")
	fmt.Println("  cocopilot migrate [up]       Apply all pending migrations")
	fmt.Println("  cocopilot migrate down       Rollback the last migration (tracking only)")
	fmt.Println("  cocopilot migrate status     Show migration status")
	fmt.Println("  cocopilot help               Show this help message")
	fmt.Println()
	fmt.Println("Worker flags:")
	fmt.Println("  --project=ID   Project ID to poll (default: proj_default)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cocopilot                           # Start server on http://127.0.0.1:8080")
	fmt.Println("  cocopilot quickstart                # Quickstart with default project")
	fmt.Println("  cocopilot worker --project=proj_abc  # Run reference worker")
	fmt.Println("  cocopilot migrate up                # Apply pending migrations")
	fmt.Println("  cocopilot migrate status            # Check migration status")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Printf("  COCO_DB_PATH   SQLite database path (default: %s)\n", defaultDBPath)
	fmt.Printf("  COCO_HTTP_ADDR HTTP listen address (default: %s)\n", defaultHTTPAddr)
	fmt.Printf("  COCO_EVENTS_RETENTION_DAYS     Events retention window in days (default: %d, 0 disables)\n", defaultEventsRetentionDays)
	fmt.Printf("  COCO_EVENTS_RETENTION_MAX_ROWS Max events to keep (default: %d, 0 disables)\n", defaultEventsRetentionMax)
	fmt.Printf("  COCO_EVENTS_PRUNE_INTERVAL_SECONDS Events prune interval in seconds (default: %d, min: %d, max: %d)\n", defaultEventsPruneIntervalSeconds, minEventsPruneIntervalSeconds, maxEventsPruneIntervalSeconds)
	fmt.Println("  COCO_NO_BROWSER                    Disable auto-opening browser on start (default: false)")
	fmt.Println("  COCO_WEBHOOK_URL                   Comma-separated webhook URLs for event notifications")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --no-browser   Disable auto-opening browser on start")
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Cannot open browser on %s; navigate manually to %s", runtime.GOOS, url)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v — navigate manually to %s", err, url)
	}
}
