package txpilot

import "time"

type Slot uint64

type Lamports uint64

type BundleID string

type Signature string

type TransactionID string

type DecisionID string

type FailureID string

type Commitment string

const (
	CommitmentProcessed Commitment = "processed"
	CommitmentConfirmed Commitment = "confirmed"
	CommitmentFinalized Commitment = "finalized"
)

type SubmissionKind string

const (
	SubmissionTransaction SubmissionKind = "transaction"
	SubmissionBundle      SubmissionKind = "bundle"
)

type Encoding string

const (
	EncodingBase64 Encoding = "base64"
	EncodingBase58 Encoding = "base58"
)

const MaxBundleTransactions = 5

type LifecycleStage string

const (
	StageCreated   LifecycleStage = "created"
	StageSubmitted LifecycleStage = "submitted"
	StageProcessed LifecycleStage = "processed"
	StageConfirmed LifecycleStage = "confirmed"
	StageFinalized LifecycleStage = "finalized"
	StageFailed    LifecycleStage = "failed"
)

type FailureKind string

const (
	FailureExpiredBlockhash  FailureKind = "expired_blockhash"
	FailureTipBelowFloor     FailureKind = "tip_below_floor"
	FailureComputeExceeded   FailureKind = "compute_exceeded"
	FailureBundleRejected    FailureKind = "bundle_rejected"
	FailureLeaderSkipped     FailureKind = "leader_skipped"
	FailureStreamGap         FailureKind = "stream_gap"
	FailureRPCError          FailureKind = "rpc_error"
	FailureInsufficientFunds FailureKind = "insufficient_funds"
	FailureInstructionError  FailureKind = "instruction_error"
	FailureDropped           FailureKind = "dropped"
	FailureUnknown           FailureKind = "unknown"
)

type PolicyMode string

const (
	ModeFast       PolicyMode = "FAST"
	ModeSafe       PolicyMode = "SAFE"
	ModeCheap      PolicyMode = "CHEAP"
	ModeAggressive PolicyMode = "AGGRESSIVE"
)

type TipMode string

const (
	TipModeAuto TipMode = "auto"
)

type TipSource string

const (
	TipSourceCaller       TipSource = "caller"
	TipSourceAuto         TipSource = "auto"
	TipSourceFloorClamped TipSource = "floor_clamped"
)

type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusSubmitted  TransactionStatus = "submitted"
	StatusProcessing TransactionStatus = "processing"
	StatusConfirmed  TransactionStatus = "confirmed"
	StatusFinalized  TransactionStatus = "finalized"
	StatusFailed     TransactionStatus = "failed"
)

type TipResolution struct {
	RequestedTipLamports Lamports  `json:"requested_tip_lamports"`
	FloorLamports        Lamports  `json:"floor_lamports"`
	FinalTipLamports     Lamports  `json:"final_tip_lamports"`
	TipSource            TipSource `json:"tip_source"`
}

// SubmitTransactionRequest forwards a single pre-signed transaction to Jito sendTransaction.
// The server auto-wraps it into a bundle with a dynamically-priced tip transaction.
type SubmitTransactionRequest struct {
	Transaction string     `json:"transaction"`
	Encoding    string     `json:"encoding,omitempty"`
	PolicyMode  PolicyMode `json:"policy_mode,omitempty"`
	TipLamports *uint64    `json:"tip_lamports,omitempty"`
	Memo        string     `json:"memo,omitempty"`
}

// SubmitBundleRequest forwards 1-5 pre-signed transactions to Jito sendBundle.
// The server appends a dynamically-priced tip transaction when room allows.
type SubmitBundleRequest struct {
	Transactions []string   `json:"transactions"`
	Encoding     string     `json:"encoding,omitempty"`
	PolicyMode   PolicyMode `json:"policy_mode,omitempty"`
	TipLamports  *uint64    `json:"tip_lamports,omitempty"`
	Memo         string     `json:"memo,omitempty"`
}

// SubmitOpsRequest triggers a server-signed operational self-transfer bundled with a dynamic tip.
type SubmitOpsRequest struct {
	PolicyMode             PolicyMode `json:"policy_mode,omitempty"`
	TipLamports            *uint64    `json:"tip_lamports,omitempty"`
	Lamports               uint64     `json:"lamports,omitempty"`
	Memo                   string     `json:"memo,omitempty"`
	InjectExpiredBlockhash bool       `json:"inject_expired_blockhash,omitempty"`
}

// SubmitResponse is returned for both single and bundle submissions.
type SubmitResponse struct {
	TransactionID    TransactionID  `json:"transaction_id"`
	SubmissionKind   SubmissionKind `json:"submission_kind"`
	Result           string         `json:"result"`
	Signatures       []string       `json:"signatures,omitempty"`
	Signature        Signature      `json:"signature,omitempty"`
	BundleID         BundleID       `json:"bundle_id,omitempty"`
	Status           string         `json:"status"`
	Encoding         string         `json:"encoding"`
	TipFloorLamports uint64         `json:"tip_floor_lamports,omitempty"`
	TipLamports      uint64         `json:"tip_lamports,omitempty"`
	TipSource        TipSource      `json:"tip_source,omitempty"`
}

type LifecycleEvent struct {
	ID            string         `json:"id"`
	TransactionID TransactionID  `json:"transaction_id"`
	Signature     Signature      `json:"signature,omitempty"`
	BundleID      BundleID       `json:"bundle_id,omitempty"`
	Stage         LifecycleStage `json:"stage"`
	Slot          Slot           `json:"slot,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
	LatencyMS     *int64         `json:"latency_ms,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type AgentDecision struct {
	DecisionID    DecisionID     `json:"decision_id"`
	TransactionID TransactionID  `json:"transaction_id,omitempty"`
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Inputs        map[string]any `json:"inputs"`
	Action        map[string]any `json:"action"`
	ConfidencePct int            `json:"confidence_pct"`
	CreatedAt     time.Time      `json:"created_at"`
}

const OpsMemoPrefix = "tx-pilot-ops:"

type LifecycleLogEntry struct {
	TransactionID         TransactionID  `json:"transaction_id"`
	BundleID              BundleID       `json:"bundle_id,omitempty"`
	Signature             Signature      `json:"signature,omitempty"`
	SubmissionKind        SubmissionKind `json:"submission_kind"`
	Status                string         `json:"status"`
	Stage                 LifecycleStage `json:"stage"`
	SubmittedSlot         uint64         `json:"submitted_slot,omitempty"`
	ProcessedSlot         uint64         `json:"processed_slot,omitempty"`
	ConfirmedSlot         uint64         `json:"confirmed_slot,omitempty"`
	FinalizedSlot         uint64         `json:"finalized_slot,omitempty"`
	SubmittedAt           *time.Time     `json:"submitted_at,omitempty"`
	ProcessedAt           *time.Time     `json:"processed_at,omitempty"`
	ConfirmedAt           *time.Time     `json:"confirmed_at,omitempty"`
	FinalizedAt           *time.Time     `json:"finalized_at,omitempty"`
	FailedAt              *time.Time     `json:"failed_at,omitempty"`
	TipLamports           int64          `json:"tip_lamports"`
	Leader                string         `json:"leader,omitempty"`
	RetryAttempt          int32          `json:"retry_attempt"`
	FailureKind           FailureKind    `json:"failure_kind,omitempty"`
	FailureTitle          string         `json:"failure_title,omitempty"`
	LatencySubmittedMS    *int64         `json:"latency_submitted_ms,omitempty"`
	LatencyProcessedMS    *int64         `json:"latency_processed_ms,omitempty"`
	LatencyConfirmedMS    *int64         `json:"latency_confirmed_ms,omitempty"`
	CommitmentProgression []string       `json:"commitment_progression"`
}
