package scheduler

import (
	"time"

	"github.com/mira4sol/aegis/internal/stream"
	"github.com/mira4sol/aegis/pkg/aegis"
)

type Decision struct {
	SubmitNow  bool
	DelaySlots uint64
	TargetSlot uint64
	Leader     string
	PolicyMode aegis.PolicyMode
	Reason     string
}

type Scheduler struct {
	policyMode aegis.PolicyMode
}

func New(policyMode aegis.PolicyMode) *Scheduler {
	return &Scheduler{policyMode: policyMode}
}

func (s *Scheduler) Evaluate(slotState *stream.SlotState, mode aegis.PolicyMode) Decision {
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
			case aegis.ModeSafe, aegis.ModeCheap:
				decision.SubmitNow = false
				decision.DelaySlots = 1
				decision.TargetSlot = currentSlot + 1
				decision.Reason = "previous leader skipped slot; delaying one slot"
			}
		}
	}

	if mode == aegis.ModeAggressive {
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
