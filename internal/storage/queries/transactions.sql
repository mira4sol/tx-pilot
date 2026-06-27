-- name: CreateTransaction :one
INSERT INTO transactions (
    id, status, stage, policy_mode, tip_lamports, requested_tip_lamports,
    floor_lamports, tip_source, memo, retry_attempt,
    submission_kind, encoding, signatures, tx_count, signature, bundle_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = $1;

-- name: GetTransactionBySignature :one
SELECT * FROM transactions WHERE signature = $1;

-- name: UpdateTransactionStatus :one
UPDATE transactions SET
    status = $2,
    stage = $3,
    signature = COALESCE($4, signature),
    bundle_id = COALESCE($5, bundle_id),
    tip_lamports = COALESCE($6, tip_lamports),
    target_slot = COALESCE($7, target_slot),
    submitted_slot = COALESCE($8, submitted_slot),
    processed_slot = COALESCE($9, processed_slot),
    confirmed_slot = COALESCE($10, confirmed_slot),
    finalized_slot = COALESCE($11, finalized_slot),
    leader = COALESCE($12, leader),
    retry_attempt = COALESCE($13, retry_attempt),
    failure_kind = COALESCE($14, failure_kind),
    agent_decision_id = COALESCE($15, agent_decision_id),
    submitted_at = COALESCE($16, submitted_at),
    processed_at = COALESCE($17, processed_at),
    confirmed_at = COALESCE($18, confirmed_at),
    finalized_at = COALESCE($19, finalized_at),
    failed_at = COALESCE($20, failed_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListRecentTransactions :many
SELECT * FROM transactions ORDER BY created_at DESC LIMIT $1;

-- name: InsertLifecycleEvent :one
INSERT INTO lifecycle_events (
    id, transaction_id, signature, bundle_id, stage, slot, latency_ms, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListLifecycleEvents :many
SELECT * FROM lifecycle_events WHERE transaction_id = $1 ORDER BY created_at ASC;

-- name: InsertAgentDecision :one
INSERT INTO agent_decisions (
    id, transaction_id, decision_type, title, summary, inputs, action, confidence_pct
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListRecentAgentDecisions :many
SELECT * FROM agent_decisions ORDER BY created_at DESC LIMIT $1;

-- name: InsertFailure :one
INSERT INTO failures (
    id, transaction_id, bundle_id, kind, title, severity, slot, recommended_action, evidence
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListRecentFailures :many
SELECT * FROM failures ORDER BY created_at DESC LIMIT $1;

-- name: InsertRecoveryAction :one
INSERT INTO recovery_actions (id, transaction_id, decision_id, label, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRecoveryActionStatus :one
UPDATE recovery_actions SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: ListRecoveryActions :many
SELECT * FROM recovery_actions ORDER BY created_at DESC LIMIT $1;

-- name: CreateBundle :one
INSERT INTO bundles (id, transaction_id, status, tip_lamports, target_slot, submitted_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateBundleStatus :one
UPDATE bundles SET status = $2, landed_at = COALESCE($3, landed_at), failed_at = COALESCE($4, failed_at)
WHERE id = $1 RETURNING *;

-- name: GetBundle :one
SELECT * FROM bundles WHERE id = $1;

-- name: ListBundles :many
SELECT * FROM bundles ORDER BY created_at DESC LIMIT $1;

-- name: CountTransactionsByStage :many
SELECT stage, COUNT(*)::bigint AS count FROM transactions GROUP BY stage;

-- name: CountFailuresToday :one
SELECT COUNT(*)::bigint FROM failures WHERE created_at >= CURRENT_DATE;

-- name: CountRetries :one
SELECT COALESCE(SUM(retry_attempt), 0)::bigint FROM transactions;

-- name: CountBundleMetrics :one
SELECT
    COUNT(*)::bigint AS sent,
    COUNT(*) FILTER (WHERE status = 'landed')::bigint AS landed,
    COUNT(*) FILTER (WHERE status IN ('pending', 'submitted'))::bigint AS pending
FROM bundles;

-- name: AvgTipLamports :one
SELECT COALESCE(AVG(tip_lamports), 0)::float8 FROM bundles WHERE tip_lamports > 0;

-- name: InsertChartPoint :exec
INSERT INTO chart_points (series, point_time, payload) VALUES ($1, $2, $3);

-- name: ListChartPoints :many
SELECT * FROM chart_points WHERE series = $1 AND point_time >= $2 ORDER BY point_time ASC;

-- name: AvgProcessedToConfirmedMS :one
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (confirmed_at - processed_at)) * 1000), 0)::float8
FROM transactions WHERE processed_at IS NOT NULL AND confirmed_at IS NOT NULL AND confirmed_at >= NOW() - INTERVAL '15 minutes';
