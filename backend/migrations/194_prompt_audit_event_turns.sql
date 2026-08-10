-- Preserve the two runtime audit inputs independently for administrator review.
-- They remain event-only fields and are excluded from list queries.
ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS latest_user_input TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS previous_assistant_output TEXT NOT NULL DEFAULT '';
