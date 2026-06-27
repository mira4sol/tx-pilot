package app

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/mira4sol/aegis/internal/agent"
	"github.com/mira4sol/aegis/internal/bundle"
	"github.com/mira4sol/aegis/internal/tx"
	"github.com/mira4sol/aegis/pkg/aegis"
	"go.uber.org/zap"
)

type tipPlan struct {
	Lamports      uint64
	FloorLamports uint64
	Source        aegis.TipSource
	Decision      agent.Decision
	DecisionID    string
	TargetSlot    uint64
	Leader        string
}

func (cp *ControlPlane) planTip(ctx context.Context, txID aegis.TransactionID, policyMode aegis.PolicyMode, requested *uint64) (tipPlan, error) {
	if cp.jito.TipAccountsCount() == 0 {
		_ = cp.jito.LoadTipAccounts(ctx)
	}
	congestion := cp.slotState.CongestionScore()
	slot := cp.slotState.CurrentSlot()
	leader := cp.slotState.LeaderAt(slot)

	recommend, err := cp.tip.Recommend(ctx, bundle.RecommendInput{
		ResolveInput: bundle.ResolveInput{
			RequestedTip:  requested,
			PolicyMode:    policyMode,
			Congestion:    congestion,
			LeaderQuality: leaderQuality(leader),
		},
		LeaderQuality: leaderQuality(leader),
	})
	if err != nil {
		return tipPlan{}, err
	}

	baseTip := uint64(recommend.Resolution.FinalTipLamports)
	landingRate := cp.landingRate(ctx)

	var decision agent.Decision
	if cp.agent != nil {
		decision, _ = cp.agent.DecideTip(ctx, agent.TipFacts{
			TransactionID: string(txID),
			PolicyMode:    policyMode,
			CongestionPct: congestion * 100,
			FloorLamports: uint64(recommend.Resolution.FloorLamports),
			BaseTip:       baseTip,
			Leader:        leader,
			LeaderQuality: leaderQuality(leader),
			LandingRate:   landingRate,
		})
	} else {
		decision = agent.Decision{
			Type: "tip_intelligence", Title: "Tip from floor",
			Summary:       fmt.Sprintf("Using %d lamports at %.0f%% congestion", baseTip, congestion*100),
			Action:        map[string]any{"kind": "set_tip", "tip_lamports": baseTip},
			ConfidencePct: 70,
		}
	}

	finalTip := tipFromDecision(decision, baseTip)
	if finalTip < uint64(recommend.Resolution.FloorLamports) {
		finalTip = uint64(recommend.Resolution.FloorLamports)
	}

	// The agent decision is persisted by the caller via commitTipDecision AFTER
	// the transaction row exists, otherwise the agent_decisions foreign key to
	// transactions is violated and the AI decision is silently dropped.
	return tipPlan{
		Lamports:      finalTip,
		FloorLamports: uint64(recommend.Resolution.FloorLamports),
		Source:        recommend.Resolution.TipSource,
		Decision:      decision,
		TargetSlot:    slot,
		Leader:        leader,
	}, nil
}

// commitTipDecision persists the agent decision produced by planTip and links it
// to the (now existing) transaction row, broadcasting it to the AI feed. It must
// be called after CreateTransaction.
func (cp *ControlPlane) commitTipDecision(ctx context.Context, txID aegis.TransactionID, plan *tipPlan) {
	if plan == nil {
		return
	}
	decisionID, err := cp.persistAgentDecision(ctx, txID, plan.Decision)
	if err != nil {
		cp.logger.Warn("tip decision not persisted",
			zap.String("transaction_id", string(txID)),
			zap.Error(err),
		)
		return
	}
	plan.DecisionID = decisionID
}

func (cp *ControlPlane) buildSignedTipTx(ctx context.Context, tipLamports uint64, blockhash solana.Hash, enc aegis.Encoding) (string, aegis.Encoding, error) {
	if enc == "" {
		enc = aegis.EncodingBase64
	}
	if cp.factory == nil {
		return "", "", fmt.Errorf("server signer not configured")
	}
	tipAccount, err := cp.jito.PickTipAccount()
	if err != nil {
		return "", "", err
	}
	tipTx, err := cp.factory.BuildTipTransfer(tipAccount, tipLamports, blockhash)
	if err != nil {
		return "", "", err
	}
	encoded, err := tx.EncodeTransaction(tipTx, enc)
	if err != nil {
		return "", "", err
	}
	return encoded, enc, nil
}

func (cp *ControlPlane) fetchProcessedBlockhash(ctx context.Context) (solana.Hash, error) {
	bh, err := cp.rpc.GetLatestBlockhash(ctx, "processed")
	if err != nil {
		return solana.Hash{}, err
	}
	return solana.MustHashFromBase58(bh.Value.Blockhash), nil
}

func leaderQuality(leader string) string {
	if leader == "" {
		return "unknown"
	}
	if containsJito(leader) {
		return "jito"
	}
	return "good"
}

func containsJito(leader string) bool {
	return leader != "" && (len(leader) > 4)
}

func (cp *ControlPlane) landingRate(ctx context.Context) float64 {
	counts, err := cp.q.CountBundleMetrics(ctx)
	if err != nil || counts.Sent == 0 {
		return 85
	}
	return float64(counts.Landed) / float64(counts.Sent) * 100
}
