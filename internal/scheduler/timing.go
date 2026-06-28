package scheduler

import (
	"time"

	"github.com/mira4sol/tx-pilot/internal/stream"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

type Decision struct {
	SubmitNow  bool
	DelaySlots uint64
	TargetSlot uint64
	Leader     string
	PolicyMode txpilot.PolicyMode
	Reason     string
}

type Scheduler struct {
	policyMode txpilot.PolicyMode
}

func New(policyMode txpilot.PolicyMode) *Scheduler {
	return &Scheduler{policyMode: policyMode}
}

func (s *Scheduler) Evaluate(slotState *stream.SlotState, mode txpilot.PolicyMode) Decision {
	currentSlot, _, slots, leaders, _, _ := slotState.Snapshot()
	if mode == "" {
		mode = s.policyMode
	}
	leader := leaders[currentSlot]
	decision := Decision{
		SubmitNow:  true,
		TargetSlot: currentSlot,
		Leader:     leader,
		PolicyMode: mode,
		Reason:     "default submit window",
	}

	for _, entry := range slots {
		if entry.Skipped && entry.Slot+1 == currentSlot {
			switch mode {
			case txpilot.ModeSafe, txpilot.ModeCheap:
				decision.SubmitNow = false
				decision.DelaySlots = 1
				decision.TargetSlot = currentSlot + 1
				decision.Reason = "previous leader skipped slot; delaying one slot"
			}
		}
	}

	if mode == txpilot.ModeAggressive {
		decision.SubmitNow = true
		decision.Reason = "aggressive mode submits immediately"
	}

	return decision
}

func (s *Scheduler) WaitDuration(delaySlots uint64) time.Duration {
	if delaySlots == 0 {
		return 0
	}
	return time.Duration(delaySlots) * 400 * time.Millisecond
}
