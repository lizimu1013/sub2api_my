-- Add a display-only multiplier for user-facing group presentation.
-- The real billing multiplier remains groups.rate_multiplier.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS display_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN groups.display_rate_multiplier IS '展示倍率，仅用于用户侧展示，不参与真实计费计算';
