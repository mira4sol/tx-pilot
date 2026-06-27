-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    signature TEXT,
    bundle_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    stage TEXT NOT NULL DEFAULT 'created',
    policy_mode TEXT NOT NULL DEFAULT 'SAFE',
    tip_lamports BIGINT NOT NULL DEFAULT 0,
    requested_tip_lamports BIGINT NOT NULL DEFAULT 0,
    floor_lamports BIGINT NOT NULL DEFAULT 0,
    tip_source TEXT NOT NULL DEFAULT 'auto',
    target_slot BIGINT,
    submitted_slot BIGINT,
    processed_slot BIGINT,
    confirmed_slot BIGINT,
    finalized_slot BIGINT,
    leader TEXT,
    retry_attempt INT NOT NULL DEFAULT 0,
    failure_kind TEXT,
    agent_decision_id TEXT,
    memo TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    finalized_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lifecycle_events (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    signature TEXT,
    bundle_id TEXT,
    stage TEXT NOT NULL,
    slot BIGINT,
    latency_ms BIGINT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lifecycle_events_tx ON lifecycle_events(transaction_id, created_at);

CREATE TABLE IF NOT EXISTS agent_decisions (
    id TEXT PRIMARY KEY,
    transaction_id TEXT REFERENCES transactions(id) ON DELETE SET NULL,
    decision_type TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    inputs JSONB NOT NULL DEFAULT '{}',
    action JSONB NOT NULL DEFAULT '{}',
    confidence_pct INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_decisions_created ON agent_decisions(created_at DESC);

CREATE TABLE IF NOT EXISTS failures (
    id TEXT PRIMARY KEY,
    transaction_id TEXT REFERENCES transactions(id) ON DELETE SET NULL,
    bundle_id TEXT,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'warning',
    slot BIGINT,
    recommended_action TEXT,
    evidence JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_failures_created ON failures(created_at DESC);

CREATE TABLE IF NOT EXISTS recovery_actions (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    decision_id TEXT REFERENCES agent_decisions(id) ON DELETE SET NULL,
    label TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bundles (
    id TEXT PRIMARY KEY,
    transaction_id TEXT REFERENCES transactions(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    tip_lamports BIGINT NOT NULL DEFAULT 0,
    target_slot BIGINT,
    submitted_at TIMESTAMPTZ,
    landed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chart_points (
    id BIGSERIAL PRIMARY KEY,
    series TEXT NOT NULL,
    point_time TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chart_points_series_time ON chart_points(series, point_time DESC);

CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    idempotency_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    response_code INT,
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS chart_points;
DROP TABLE IF EXISTS bundles;
DROP TABLE IF EXISTS recovery_actions;
DROP TABLE IF EXISTS failures;
DROP TABLE IF EXISTS agent_decisions;
DROP TABLE IF EXISTS lifecycle_events;
DROP TABLE IF EXISTS transactions;
