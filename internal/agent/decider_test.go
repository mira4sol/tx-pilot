package agent_test

import (
	"context"
	"testing"

	"github.com/mira4sol/aegis/internal/agent"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func TestFallbackDecisionExpiredBlockhash(t *testing.T) {
	fake := &agent.FakeAgent{}
	decision, err := fake.Decide(context.Background(), agent.DecisionFacts{
		Failure: failure.Classification{Kind: aegis.FailureExpiredBlockhash},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action["kind"] != "refresh_blockhash" {
		t.Fatalf("unexpected action: %v", decision.Action)
	}
}
