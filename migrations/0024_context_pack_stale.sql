-- 0024_context_pack_stale.sql
-- Purpose: Add stale flag to context_packs so workers can mark packs as
-- outdated after repo changes, prompting regeneration before next claim.

ALTER TABLE context_packs ADD COLUMN stale INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_context_packs_stale ON context_packs(project_id, stale);
