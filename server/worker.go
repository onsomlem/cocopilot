// Worker — thin wrapper over internal/worker.
package server

import "github.com/onsomlem/cocopilot/internal/worker"

type TaskExecutor = worker.TaskExecutor
type PlaceholderExecutor = worker.PlaceholderExecutor
type ScriptExecutor = worker.ScriptExecutor
type WebhookExecutor = worker.WebhookExecutor

func resolveExecutor() TaskExecutor { return worker.ResolveExecutor() }
func runWorker(projectID string) error { return worker.RunWorker(projectID) }
