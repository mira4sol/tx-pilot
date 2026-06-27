package failure

import (
	"strings"

	"github.com/mira4sol/aegis/pkg/aegis"
)

// ClassifyOnChain interprets the `err` object returned by getSignatureStatuses
// for a transaction that landed but failed execution. The raw JSON is always
// preserved as evidence so the true on-chain reason is never lost.
func ClassifyOnChain(rawErr string, evidence Evidence) Classification {
	if evidence.Extra == nil {
		evidence.Extra = map[string]any{}
	}
	evidence.Extra["on_chain_error"] = rawErr
	s := strings.ToLower(rawErr)
	switch {
	// System program ResultWithNegativeLamports / SPL TokenError::InsufficientFunds
	// both surface as Custom(1) on a transfer instruction, plus rent shortfalls.
	case strings.Contains(s, "custom\":1}") || strings.Contains(s, "custom\": 1}"),
		strings.Contains(s, "insufficientfunds"),
		strings.Contains(s, "insufficientfundsforrent"):
		return Classification{
			Kind: aegis.FailureInsufficientFunds, Title: "Insufficient funds", Severity: "error",
			RecommendedAction: "Fund the account (native SOL or the SPL token being transferred) and resubmit",
			Evidence:          evidence,
		}
	case strings.Contains(s, "computebudgetexceeded"), strings.Contains(s, "exceeded cus"), strings.Contains(s, "computationalbudgetexceeded"):
		return Classification{
			Kind: aegis.FailureComputeExceeded, Title: "Compute budget exceeded", Severity: "error",
			RecommendedAction: "Raise the compute unit limit and resubmit", Evidence: evidence,
		}
	case strings.Contains(s, "instructionerror"):
		return Classification{
			Kind: aegis.FailureInstructionError, Title: "Instruction failed on-chain", Severity: "error",
			RecommendedAction: "Inspect the on-chain error and fix the instruction inputs", Evidence: evidence,
		}
	default:
		return Classification{
			Kind: aegis.FailureInstructionError, Title: "Transaction failed on-chain", Severity: "error",
			RecommendedAction: "Inspect the on-chain error", Evidence: evidence,
		}
	}
}

type Evidence struct {
	RPCMessage   string         `json:"rpc_message,omitempty"`
	BundleStatus string         `json:"bundle_status,omitempty"`
	Slot         uint64         `json:"slot,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
}

type Classification struct {
	Kind              aegis.FailureKind
	Title             string
	Severity          string
	RecommendedAction string
	Evidence          Evidence
}

func Classify(err error, evidence Evidence) Classification {
	msg := ""
	if err != nil {
		msg = strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(msg, "blockhash not found"), strings.Contains(msg, "blockhash expired"), strings.Contains(msg, "block height exceeded"):
		return Classification{
			Kind: aegis.FailureExpiredBlockhash, Title: "Blockhash expired", Severity: "warning",
			RecommendedAction: "Refresh blockhash and resubmit", Evidence: evidence,
		}
	case strings.Contains(msg, "insufficient"), strings.Contains(msg, "tip"), strings.Contains(msg, "priority fee"):
		return Classification{
			Kind: aegis.FailureTipBelowFloor, Title: "Tip below floor", Severity: "warning",
			RecommendedAction: "Increase tip above dynamic floor", Evidence: evidence,
		}
	case strings.Contains(msg, "compute"), strings.Contains(msg, "exceeded cus"):
		return Classification{
			Kind: aegis.FailureComputeExceeded, Title: "Compute budget exceeded", Severity: "error",
			RecommendedAction: "Raise compute unit limit and retry", Evidence: evidence,
		}
	case strings.Contains(msg, "bundle"), strings.Contains(msg, "rejected"):
		return Classification{
			Kind: aegis.FailureBundleRejected, Title: "Bundle rejected by Jito", Severity: "warning",
			RecommendedAction: "Recalculate tip and retry", Evidence: evidence,
		}
	case strings.Contains(msg, "skipped"):
		return Classification{
			Kind: aegis.FailureLeaderSkipped, Title: "Leader skipped slot", Severity: "warning",
			RecommendedAction: "Wait for next leader window", Evidence: evidence,
		}
	case strings.Contains(msg, "stream"), strings.Contains(msg, "gap"):
		return Classification{
			Kind: aegis.FailureStreamGap, Title: "Stream gap detected", Severity: "warning",
			RecommendedAction: "Reconnect stream and verify status", Evidence: evidence,
		}
	case strings.Contains(msg, "rpc"):
		return Classification{
			Kind: aegis.FailureRPCError, Title: "RPC error", Severity: "error",
			RecommendedAction: "Retry with fresh RPC evidence", Evidence: evidence,
		}
	default:
		return Classification{
			Kind: aegis.FailureUnknown, Title: "Unknown failure", Severity: "error",
			RecommendedAction: "Inspect lifecycle timeline", Evidence: evidence,
		}
	}
}
