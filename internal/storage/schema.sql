CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    signature TEXT,
    bundle_id TEXT,
    submission_kind TEXT NOT NULL DEFAULT 'transaction',
    encoding TEXT NOT NULL DEFAULT 'base64',
    signatures JSONB NOT NULL DEFAULT '[]',
    tx_count INT NOT NULL DEFAULT 1,
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
