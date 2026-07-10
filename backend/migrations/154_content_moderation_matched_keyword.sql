-- Store the exact keyword that caused a keyword_block audit record.

ALTER TABLE content_moderation_logs
    ADD COLUMN IF NOT EXISTS matched_keyword TEXT NOT NULL DEFAULT '';
