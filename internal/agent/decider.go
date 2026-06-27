package agent

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/pkg/aegis"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type Provider interface {
	Decide(ctx context.Context, facts DecisionFacts) (Decision, error)
	DecideTip(ctx context.Context, facts TipFacts) (Decision, error)
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
	logger *zap.Logger
}

func NewOpenAIAgent(cfg *config.Config) *OpenAIAgent {
	return &OpenAIAgent{client: openai.NewClient(cfg.OpenAIAPIKey), model: cfg.OpenAIModel}
}

func (a *OpenAIAgent) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

func (a *OpenAIAgent) Decide(ctx context.Context, facts DecisionFacts) (Decision, error) {
	start := time.Now()
	prompt := buildDecisionPrompt(facts)

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You make concise operational decisions for Solana transaction recovery."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		dec := fallbackDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "failure_recovery", TransactionID: facts.TransactionID, Prompt: prompt,
			Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
			Inputs: map[string]any{"error": err.Error()},
		})
		return dec, nil
	}
	if len(resp.Choices) == 0 {
		dec := fallbackDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "failure_recovery", TransactionID: facts.TransactionID, Prompt: prompt,
			Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	content := normalizeModelJSON(resp.Choices[0].Message.Content)
	var out struct {
		Type          string         `json:"type"`
		Title         string         `json:"title"`
		Summary       string         `json:"summary"`
		ConfidencePct int            `json:"confidence_pct"`
		Action        map[string]any `json:"action"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		dec := fallbackDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "failure_recovery", TransactionID: facts.TransactionID, Prompt: prompt,
			RawOutput: content, Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	if out.Action == nil || out.Type == "" {
		dec := fallbackDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "failure_recovery", TransactionID: facts.TransactionID, Prompt: prompt,
			RawOutput: content, Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	dec := Decision{
		Type: out.Type, Title: out.Title, Summary: out.Summary,
		Inputs: map[string]any{
			"failure_kind":   string(facts.Failure.Kind),
			"retry_attempt":  facts.RetryAttempt,
			"congestion_pct": facts.CongestionPct,
			"slot":           facts.CurrentSlot,
		},
		Action: out.Action, ConfidencePct: out.ConfidencePct,
	}
	a.recordTrace(ReasoningTrace{
		Kind: "failure_recovery", TransactionID: facts.TransactionID, Prompt: prompt,
		RawOutput: content, Decision: dec, UsedFallback: false, LatencyMS: elapsedMS(start),
	})
	return dec, nil
}

type TipFacts struct {
	TransactionID string
	PolicyMode    aegis.PolicyMode
	CongestionPct float64
	FloorLamports uint64
	BaseTip       uint64
	Leader        string
	LeaderQuality string
	LandingRate   float64
}

func (a *OpenAIAgent) DecideTip(ctx context.Context, facts TipFacts) (Decision, error) {
	start := time.Now()
	prompt := buildTipPrompt(facts)

	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You optimize Solana Jito bundle tips for landing probability under cost constraints."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		dec := fallbackTipDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "tip_intelligence", TransactionID: facts.TransactionID, Prompt: prompt,
			Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
			Inputs: map[string]any{"error": err.Error()},
		})
		return dec, nil
	}
	if len(resp.Choices) == 0 {
		dec := fallbackTipDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "tip_intelligence", TransactionID: facts.TransactionID, Prompt: prompt,
			Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	content := normalizeModelJSON(resp.Choices[0].Message.Content)
	var out struct {
		Type          string         `json:"type"`
		Title         string         `json:"title"`
		Summary       string         `json:"summary"`
		ConfidencePct int            `json:"confidence_pct"`
		Action        map[string]any `json:"action"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		dec := fallbackTipDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "tip_intelligence", TransactionID: facts.TransactionID, Prompt: prompt,
			RawOutput: content, Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	if out.Action == nil {
		dec := fallbackTipDecision(facts)
		a.recordTrace(ReasoningTrace{
			Kind: "tip_intelligence", TransactionID: facts.TransactionID, Prompt: prompt,
			RawOutput: content, Decision: dec, UsedFallback: true, LatencyMS: elapsedMS(start),
		})
		return dec, nil
	}
	if _, ok := out.Action["tip_lamports"]; !ok {
		out.Action["kind"] = "set_tip"
		out.Action["tip_lamports"] = float64(facts.BaseTip)
	}
	if out.Type == "" {
		out.Type = "tip_intelligence"
	}
	dec := Decision{
		Type: out.Type, Title: out.Title, Summary: out.Summary,
		Inputs: map[string]any{
			"congestion_pct":     facts.CongestionPct,
			"floor_tip_lamports": facts.FloorLamports,
			"base_tip_lamports":  facts.BaseTip,
			"leader":             facts.Leader,
		},
		Action: out.Action, ConfidencePct: out.ConfidencePct,
	}
	a.recordTrace(ReasoningTrace{
		Kind: "tip_intelligence", TransactionID: facts.TransactionID, Prompt: prompt,
		RawOutput: content, Decision: dec, UsedFallback: false, LatencyMS: elapsedMS(start),
	})
	return dec, nil
}

func normalizeModelJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
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

func (f *FakeAgent) DecideTip(ctx context.Context, facts TipFacts) (Decision, error) {
	if f.Err != nil {
		return Decision{}, f.Err
	}
	return fallbackTipDecision(facts), nil
}

func Now() time.Time { return time.Now().UTC() }
