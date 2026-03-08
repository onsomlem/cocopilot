package server

import (
	"database/sql"
	"time"

	"github.com/onsomlem/cocopilot/internal/notifications"
)

var webhookNotifier = notifications.Notifier
var notifiableEvents = notifications.NotifiableEvents

func initWebhookNotifier() { notifications.InitWebhookNotifier() }

func NotifyEvent(event Event) { notifications.NotifyEvent(event) }

func DetectStalledTasks(database *sql.DB, threshold time.Duration) {
	notifications.DetectStalledTasks(database, threshold)
}

func DetectIdleProjects(database *sql.DB, threshold time.Duration) {
	notifications.DetectIdleProjects(database, threshold)
}
