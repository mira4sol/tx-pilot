-- +goose Up
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS submission_kind TEXT NOT NULL DEFAULT 'transaction',
    ADD COLUMN IF NOT EXISTS encoding TEXT NOT NULL DEFAULT 'base64',
    ADD COLUMN IF NOT EXISTS signatures JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS tx_count INT NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE transactions
    DROP COLUMN IF EXISTS submission_kind,
    DROP COLUMN IF EXISTS encoding,
    DROP COLUMN IF EXISTS signatures,
    DROP COLUMN IF EXISTS tx_count;
