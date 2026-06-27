package app

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mira4sol/aegis/internal/agent"
	"github.com/mira4sol/aegis/internal/storage"
	"github.com/mira4sol/aegis/internal/storage/dbgen"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func (cp *ControlPlane) persistAgentDecision(ctx context.Context, txID aegis.TransactionID, decision agent.Decision) (string, error) {
	decisionID := "dec_" + uuid.NewString()
	inputs, _ := json.Marshal(decision.Inputs)
	action, _ := json.Marshal(decision.Action)
	_, err := cp.q.InsertAgentDecision(ctx, dbgen.InsertAgentDecisionParams{
		ID: decisionID, TransactionID: pgtype.Text{String: string(txID), Valid: true},
		DecisionType: decision.Type, Title: decision.Title, Summary: decision.Summary,
		Inputs: inputs, Action: action, ConfidencePct: int32(decision.ConfidencePct),
	})
	if err != nil {
		storage.LogDB(cp.logger, "InsertAgentDecision", err)
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
	storage.LogDB(cp.logger, "UpdateTransactionStatus.agent_decision_id", err)
	if cp.notify != nil {
		cp.notify.BroadcastDecision(map[string]any{
			"decision_id": decisionID, "transaction_id": txID,
			"type": decision.Type, "title": decision.Title, "summary": decision.Summary,
			"inputs": decision.Inputs, "action": decision.Action, "confidence_pct": decision.ConfidencePct,
		})
	}
	return decisionID, nil
}

func (cp *ControlPlane) persistRecoveryAction(ctx context.Context, txID aegis.TransactionID, decisionID, label, status string) {
	_, err := cp.q.InsertRecoveryAction(ctx, dbgen.InsertRecoveryActionParams{
		ID: "rec_" + uuid.NewString(), TransactionID: string(txID),
		DecisionID: pgtype.Text{String: decisionID, Valid: decisionID != ""},
		Label:      label, Status: status,
	})
	storage.LogDB(cp.logger, "InsertRecoveryAction", err)
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
