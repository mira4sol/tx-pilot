package agent_test

import (
	"context"
	"testing"

	"github.com/mira4sol/tx-pilot/internal/agent"
	"github.com/mira4sol/tx-pilot/internal/failure"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func TestFallbackDecisionExpiredBlockhash(t *testing.T) {
	fake := &agent.FakeAgent{}
	decision, err := fake.Decide(context.Background(), agent.DecisionFacts{
		Failure: failure.Classification{Kind: txpilot.FailureExpiredBlockhash},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action["kind"] != "refresh_blockhash" {
		t.Fatalf("unexpected action: %v", decision.Action)
	}
}

func TestFallbackTipDecision(t *testing.T) {
	fake := &agent.FakeAgent{}
	decision, err := fake.DecideTip(context.Background(), agent.TipFacts{
		CongestionPct: 80, BaseTip: 1000, FloorLamports: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action["kind"] != "set_tip" {
		t.Fatalf("unexpected action: %v", decision.Action)
	}
}
