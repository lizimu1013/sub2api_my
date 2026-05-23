-- 展示倍率仅用于用户界面展示，不参与实际计费。
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS display_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN groups.display_rate_multiplier IS '展示倍率，仅用于用户界面展示，不参与实际计费';
