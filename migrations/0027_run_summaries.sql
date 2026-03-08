-- Run Summaries: structured completion data extracted from task outputs.
CREATE TABLE IF NOT EXISTS run_summaries (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id        TEXT    NOT NULL,
    task_id       INTEGER NOT NULL,
    status        TEXT    NOT NULL DEFAULT 'SUCCEEDED',
    summary       TEXT    NOT NULL DEFAULT '',
    changes_made  TEXT    NOT NULL DEFAULT '[]',
    files_touched TEXT    NOT NULL DEFAULT '[]',
    commands_run  TEXT    NOT NULL DEFAULT '[]',
    tests_run     TEXT    NOT NULL DEFAULT '[]',
    risks         TEXT    NOT NULL DEFAULT '[]',
    created_at    TEXT    NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_summaries_run_id  ON run_summaries(run_id);
CREATE INDEX IF NOT EXISTS idx_run_summaries_task_id ON run_summaries(task_id);
