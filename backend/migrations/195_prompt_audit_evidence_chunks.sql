-- Keep each aggregated scanner score paired with the chunk that supplied it.
ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS scanner_evidence_chunks JSONB NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_prompt_audit_events_evidence_chunks_json'
    ) THEN
        ALTER TABLE prompt_audit_events
            ADD CONSTRAINT chk_prompt_audit_events_evidence_chunks_json
            CHECK (jsonb_typeof(scanner_evidence_chunks) = 'object');
    END IF;
END $$;
