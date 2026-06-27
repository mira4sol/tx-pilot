package notify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// IdempotencyStore prevents duplicate webhook deliveries for the same lifecycle event.
type IdempotencyStore struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{seen: make(map[string]struct{})}
}

func IdempotencyKey(eventType, transactionID, stage string) string {
	raw := fmt.Sprintf("%s:%s:%s", eventType, transactionID, stage)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *IdempotencyStore) Seen(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.seen[key]
	return ok
}

func (s *IdempotencyStore) Mark(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[key] = struct{}{}
}
