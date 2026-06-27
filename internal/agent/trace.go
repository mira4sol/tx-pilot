package agent

import (
	"time"

	"go.uber.org/zap"
)

// ReasoningTrace captures the operational reasoning path for an agent decision.
type ReasoningTrace struct {
	Kind          string         `json:"kind"`
	TransactionID string         `json:"transaction_id,omitempty"`
	Prompt        string         `json:"prompt,omitempty"`
	RawOutput     string         `json:"raw_output,omitempty"`
	Decision      Decision       `json:"decision"`
	UsedFallback  bool           `json:"used_fallback"`
	LatencyMS     int64          `json:"latency_ms"`
	Inputs        map[string]any `json:"inputs,omitempty"`
}

func (a *OpenAIAgent) recordTrace(trace ReasoningTrace) {
	if a.logger == nil {
		return
	}
	a.logger.Info("agent reasoning trace",
		zap.String("kind", trace.Kind),
		zap.String("transaction_id", trace.TransactionID),
		zap.Bool("used_fallback", trace.UsedFallback),
		zap.Int64("latency_ms", trace.LatencyMS),
		zap.String("decision_type", trace.Decision.Type),
		zap.String("decision_title", trace.Decision.Title),
		zap.String("summary", trace.Decision.Summary),
		zap.Int("confidence_pct", trace.Decision.ConfidencePct),
	)
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
