package app

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/tx-pilot/internal/agent"
	"github.com/mira4sol/tx-pilot/internal/storage"
	"github.com/mira4sol/tx-pilot/internal/storage/dbgen"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
	"go.uber.org/zap"
)

func (cp *ControlPlane) persistAgentDecision(ctx context.Context, txID txpilot.TransactionID, decision agent.Decision) (string, error) {
	decisionID := "dec_" + uuid.NewString()
	inputs, _ := json.Marshal(decision.Inputs)
	action, _ := json.Marshal(decision.Action)
	_, err := cp.q.InsertAgentDecision(ctx, dbgen.InsertAgentDecisionParams{
		ID: decisionID, TransactionID: pgtype.Text{String: string(txID), Valid: true},
		DecisionType: decision.Type, Title: decision.Title, Summary: decision.Summary,
		Inputs: inputs, Action: action, ConfidencePct: int32(decision.ConfidencePct),
	})
	if err != nil {
		storage.LogDBOp(cp.logger, "InsertAgentDecision", err,
			zap.String("transaction_id", string(txID)),
			zap.String("decision_type", decision.Type),
		)
		return "", err
	}
	row, err := cp.q.GetTransaction(ctx, string(txID))
	if err != nil {
		return decisionID, err
	}
	_, err = cp.q.UpdateTransactionStatus(ctx, dbgen.UpdateTransactionStatusParams{
		ID: string(txID), Status: row.Status, Stage: row.Stage,
		AgentDecisionID: pgtype.Text{String: decisionID, Valid: true},
	})
	storage.LogDBOp(cp.logger, "UpdateTransactionStatus.agent_decision_id", err,
		zap.String("transaction_id", string(txID)),
		zap.String("decision_id", decisionID),
	)
	if err == nil {
		cp.logger.Info("agent decision persisted",
			zap.String("transaction_id", string(txID)),
			zap.String("decision_id", decisionID),
			zap.String("decision_type", decision.Type),
			zap.String("title", decision.Title),
			zap.Int("confidence_pct", decision.ConfidencePct),
		)
	}
	if cp.notify != nil {
		cp.notify.BroadcastDecision(map[string]any{
			"decision_id": decisionID, "transaction_id": txID,
			"type": decision.Type, "title": decision.Title, "summary": decision.Summary,
			"inputs": decision.Inputs, "action": decision.Action, "confidence_pct": decision.ConfidencePct,
		})
	}
	return decisionID, nil
}

func (cp *ControlPlane) persistRecoveryAction(ctx context.Context, txID txpilot.TransactionID, decisionID, label, status string) string {
	recoveryID := "rec_" + uuid.NewString()
	_, err := cp.q.InsertRecoveryAction(ctx, dbgen.InsertRecoveryActionParams{
		ID: recoveryID, TransactionID: string(txID),
		DecisionID: pgtype.Text{String: decisionID, Valid: decisionID != ""},
		Label:      label, Status: status,
	})
	storage.LogDBOp(cp.logger, "InsertRecoveryAction", err,
		zap.String("transaction_id", string(txID)),
		zap.String("recovery_id", recoveryID),
		zap.String("status", status),
		zap.String("label", label),
	)
	return recoveryID
}

func (cp *ControlPlane) updateRecoveryActionStatus(ctx context.Context, recoveryID, status string) {
	if recoveryID == "" {
		return
	}
	_, err := cp.q.UpdateRecoveryActionStatus(ctx, dbgen.UpdateRecoveryActionStatusParams{
		ID: recoveryID, Status: status,
	})
	storage.LogDBOp(cp.logger, "UpdateRecoveryActionStatus", err,
		zap.String("recovery_id", recoveryID),
		zap.String("status", status),
	)
}

func tipFromDecision(decision agent.Decision, fallback uint64) uint64 {
	if v, ok := decision.Action["tip_lamports"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				return uint64(n)
			}
		case int:
			if n > 0 {
				return uint64(n)
			}
		case int64:
			if n > 0 {
				return uint64(n)
			}
		}
	}
	return fallback
}

func actionKind(decision agent.Decision) string {
	if v, ok := decision.Action["kind"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
