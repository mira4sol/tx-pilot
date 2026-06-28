package stream

import (
	"sync"

	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

type TrackedSignature struct {
	TransactionID txpilot.TransactionID
	BundleID      txpilot.BundleID
	Signature     txpilot.Signature
}

type SignatureTracker struct {
	mu       sync.RWMutex
	tracked  map[string]TrackedSignature
	order    []string
	onChange func()
}

func NewSignatureTracker() *SignatureTracker {
	return &SignatureTracker{tracked: make(map[string]TrackedSignature)}
}

func (t *SignatureTracker) SetOnChange(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onChange = fn
}

func (t *SignatureTracker) Track(sig string, entry TrackedSignature) {
	if sig == "" {
		return
	}
	t.mu.Lock()
	if _, exists := t.tracked[sig]; !exists {
		t.order = append(t.order, sig)
	}
	t.tracked[sig] = entry
	onChange := t.onChange
	t.mu.Unlock()
	if onChange != nil {
		onChange()
	}
}

func (t *SignatureTracker) Untrack(sig string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.tracked[sig]; !ok {
		return
	}
	delete(t.tracked, sig)
	for i, s := range t.order {
		if s == sig {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}
	if t.onChange != nil {
		t.onChange()
	}
}

func (t *SignatureTracker) Lookup(sig string) (TrackedSignature, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.tracked[sig]
	return entry, ok
}

// Signatures returns up to limit tracked signatures in registration order (most recent last).
func (t *SignatureTracker) Signatures(limit int) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if limit <= 0 || len(t.order) == 0 {
		return nil
	}
	start := 0
	if len(t.order) > limit {
		start = len(t.order) - limit
	}
	out := make([]string, len(t.order[start:]))
	copy(out, t.order[start:])
	return out
}
