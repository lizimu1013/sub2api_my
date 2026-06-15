-- Add administrator-controlled group-level first token delay.
-- 0 means no extra delay.

ALTER TABLE groups ADD COLUMN IF NOT EXISTS first_token_delay_ms integer NOT NULL DEFAULT 0;

COMMENT ON COLUMN groups.first_token_delay_ms IS '分组额外首 token 延迟（毫秒）；0 表示不延迟；仅管理员可配置。';
