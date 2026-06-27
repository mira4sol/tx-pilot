//go:build integration

package developer_test

import (
	"testing"
	"time"

	"github.com/mira4sol/aegis/test/helpers"
)

func TestPollTransactionUntilTerminal(t *testing.T) {
	submit := helpers.SubmitDeveloperTx(t, "integration-poll")

	final := helpers.PollTransaction(t, string(submit.TransactionID), 45*time.Second)
	status, _ := final["status"].(string)
	t.Logf("terminal status=%s tx_id=%s", status, submit.TransactionID)

	switch status {
	case "submitted", "processed", "confirmed", "finalized", "failed":
		// acceptable terminal or near-terminal states during live infra
	default:
		t.Fatalf("unexpected status %q", status)
	}
}
