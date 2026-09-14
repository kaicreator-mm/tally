-- +goose Up
-- P6 closure repair — make replay idempotency a database fact, not a
-- single-writer convention. A semantic occurrence is identified by the full
-- coordinate (task_id, changeset_id, event_kind, recorded_at). At least one
-- binding is required by the domain layer; the three partial unique indexes
-- cover every valid binding shape without relying on NULL equality semantics.
CREATE UNIQUE INDEX outcome_event_once_task_and_changeset
  ON outcome_event (task_id, changeset_id, event_kind, recorded_at)
  WHERE task_id IS NOT NULL AND changeset_id IS NOT NULL;

CREATE UNIQUE INDEX outcome_event_once_task_only
  ON outcome_event (task_id, event_kind, recorded_at)
  WHERE task_id IS NOT NULL AND changeset_id IS NULL;

CREATE UNIQUE INDEX outcome_event_once_changeset_only
  ON outcome_event (changeset_id, event_kind, recorded_at)
  WHERE task_id IS NULL AND changeset_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS outcome_event_once_changeset_only;
DROP INDEX IF EXISTS outcome_event_once_task_only;
DROP INDEX IF EXISTS outcome_event_once_task_and_changeset;
