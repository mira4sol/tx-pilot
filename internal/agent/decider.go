package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/pkg/aegis"
	openai "github.com/sashabaranov/go-openai"
)

type Provider interface {
	Decide(ctx context.Context, facts DecisionFacts) (Decision, error)
}

type DecisionFacts struct {
	TransactionID string
	Failure       failure.Classification
	RetryAttempt  int
	CongestionPct float64
	CurrentTip    uint64
	FloorTip      uint64
	CurrentSlot   uint64
	Leader        string
}

type Decision struct {
	Type          string
	Title         string
	Summary       string
	Inputs        map[string]any
	Action        map[string]any
	ConfidencePct int
}

type OpenAIAgent struct {
	client *openai.Client
	model  string
}

func NewOpenAIAgent(cfg *config.Config) *OpenAIAgent {
	return &OpenAIAgent{client: openai.NewClient(cfg.OpenAIAPIKey), model: cfg.OpenAIModel}
}

const decisionSchema = `Return ONLY valid JSON with keys:
type, title, summary, confidence_pct, action.
action must include kind one of: refresh_blockhash, increase_tip, delay_submission, change_mode, abort.
For increase_tip include tip_delta_pct number. For delay_submission include delay_slots number.`

func (a *OpenAIAgent) Decide(ctx context.Context, facts DecisionFacts) (Decision, error) {
	prompt := fmt.Sprintf(`You are Aegis, an autonomous Solana transaction SRE.
Analyze the failure and decide the next operational action.
Facts:
- transaction_id: %s
- failure_kind: %s
- failure_title: %s
- retry_attempt: %d
- congestion_pct: %.1f
- current_tip_lamports: %d
- floor_tip_lamports: %d
- current_slot: %d
- leader: %s
%s`, facts.TransactionID, facts.Failure.Kind, facts.Failure.Title, facts.RetryAttempt,
		facts.CongestionPct, facts.CurrentTip, facts.FloorTip, facts.CurrentSlot, facts.Leader, decisionSchema)

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You make concise operational decisions for Solana transaction recovery."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return fallbackDecision(facts), nil
	}
	if len(resp.Choices) == 0 {
		return fallbackDecision(facts), nil
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	var out struct {
		Type          string         `json:"type"`
		Title         string         `json:"title"`
		Summary       string         `json:"summary"`
		ConfidencePct int            `json:"confidence_pct"`
		Action        map[string]any `json:"action"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return fallbackDecision(facts), nil
	}
	if out.Action == nil || out.Type == "" {
		return fallbackDecision(facts), nil
	}
	return Decision{
		Type: out.Type, Title: out.Title, Summary: out.Summary,
		Inputs: map[string]any{
			"failure_kind": string(facts.Failure.Kind),
			"retry_attempt": facts.RetryAttempt,
			"congestion_pct": facts.CongestionPct,
			"slot": facts.CurrentSlot,
		},
		Action: out.Action, ConfidencePct: out.ConfidencePct,
	}, nil
}

func fallbackDecision(facts DecisionFacts) Decision {
	switch facts.Failure.Kind {
	case aegis.FailureExpiredBlockhash:
		return Decision{
			Type: "blockhash_refresh", Title: "Refresh blockhash",
			Summary: "Fallback: expired blockhash requires refresh before resubmit",
			Inputs: map[string]any{"failure_kind": string(facts.Failure.Kind)},
			Action: map[string]any{"kind": "refresh_blockhash"},
			ConfidencePct: 70,
		}
	case aegis.FailureTipBelowFloor, aegis.FailureBundleRejected:
		return Decision{
			Type: "tip_adjustment", Title: "Increase tip",
			Summary: "Fallback: raise tip above observed floor",
			Inputs: map[string]any{"floor_tip_lamports": facts.FloorTip},
			Action: map[string]any{"kind": "increase_tip", "tip_delta_pct": 15},
			ConfidencePct: 75,
		}
	default:
		return Decision{
			Type: "retry_backoff", Title: "Delay and retry",
			Summary: "Fallback: wait one leader window before retry",
			Inputs: map[string]any{"failure_kind": string(facts.Failure.Kind)},
			Action: map[string]any{"kind": "delay_submission", "delay_slots": 1},
			ConfidencePct: 60,
		}
	}
}

type FakeAgent struct {
	Decision Decision
	Err      error
}

func (f *FakeAgent) Decide(ctx context.Context, facts DecisionFacts) (Decision, error) {
	if f.Err != nil {
		return Decision{}, f.Err
	}
	if f.Decision.Type != "" {
		return f.Decision, nil
	}
	return fallbackDecision(facts), nil
}

func Now() time.Time { return time.Now().UTC() }
