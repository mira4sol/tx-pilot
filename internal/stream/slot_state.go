package stream

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/mira4sol/tx-pilot/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type GeyserClient struct {
	conn   *grpc.ClientConn
	logger *zap.Logger
	cfg    *config.Config
}

func NewGeyserClient(cfg *config.Config, logger *zap.Logger) (*GeyserClient, error) {
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	conn, err := grpc.NewClient(cfg.YellowstoneGRPCURL, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial geyser: %w", err)
	}
	return &GeyserClient{conn: conn, logger: logger, cfg: cfg}, nil
}

func (c *GeyserClient) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *GeyserClient) AuthContext(ctx context.Context) context.Context {
	if c.cfg.YellowstoneGRPCToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-token", c.cfg.YellowstoneGRPCToken)
}

func (c *GeyserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type SlotEntry struct {
	Slot       uint64    `json:"slot"`
	AgeMS      int64     `json:"age_ms"`
	Leader     string    `json:"leader"`
	Status     string    `json:"status"`
	JitoLeader bool      `json:"jito_leader"`
	Skipped    bool      `json:"skipped"`
	SeenAt     time.Time `json:"-"`
}

type SlotState struct {
	mu             sync.RWMutex
	currentSlot    uint64
	tps            float64
	leaders        map[uint64]string
	recentSlots    []SlotEntry
	lastUpdate     time.Time
	streamLagSlots int64
	reconnectCount int64
}

func NewSlotState() *SlotState {
	return &SlotState{
		leaders:     make(map[uint64]string),
		recentSlots: make([]SlotEntry, 0, 64),
	}
}

func (s *SlotState) UpdateSlot(slot uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.currentSlot > 0 && slot > s.currentSlot+1 {
		for skipped := s.currentSlot + 1; skipped < slot; skipped++ {
			s.recentSlots = append([]SlotEntry{{
				Slot: skipped, AgeMS: 0, Leader: s.leaders[skipped], Status: "skipped", Skipped: true, SeenAt: now,
			}}, s.recentSlots...)
		}
	}
	s.currentSlot = slot
	leader := s.leaders[slot]
	s.recentSlots = append([]SlotEntry{{
		Slot: slot, AgeMS: 0, Leader: leader, Status: "current", JitoLeader: isJitoLeader(leader), SeenAt: now,
	}}, s.recentSlots...)
	if len(s.recentSlots) > 64 {
		s.recentSlots = s.recentSlots[:64]
	}
	s.lastUpdate = now
}

func (s *SlotState) UpdateTPS(tps float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tps = tps
}

func (s *SlotState) UpdateLeaderSchedule(schedule map[uint64]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for slot, leader := range schedule {
		s.leaders[slot] = leader
	}
}

func (s *SlotState) CurrentSlot() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSlot
}

// LeaderAt returns the validator identity scheduled for the given slot.
func (s *SlotState) LeaderAt(slot uint64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leaders[slot]
}

// CongestionScore returns a 0-1 congestion estimate from TPS, skipped slots, and stream drift.
func (s *SlotState) CongestionScore() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tpsScore := s.tps / 5000.0
	if tpsScore > 1 {
		tpsScore = 1
	}
	skipped := 0
	for _, entry := range s.recentSlots {
		if entry.Skipped {
			skipped++
		}
	}
	skipScore := float64(skipped) / 64.0
	driftScore := 0.0
	if !s.lastUpdate.IsZero() {
		driftMS := time.Since(s.lastUpdate).Milliseconds()
		if driftMS > 800 {
			driftScore = 0.3
		} else if driftMS > 400 {
			driftScore = 0.15
		}
	}
	score := tpsScore*0.5 + skipScore*0.35 + driftScore
	if score > 1 {
		return 1
	}
	if score < 0.05 {
		return 0.05
	}
	return score
}

// CongestionPct returns congestion as an integer percentage 10-100.
func (s *SlotState) CongestionPct() int {
	pct := int(s.CongestionScore() * 100)
	if pct < 10 {
		return 10
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// SkippedSlotsInWindow counts skipped slots in the recent window.
func (s *SlotState) SkippedSlotsInWindow() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, entry := range s.recentSlots {
		if entry.Skipped {
			n++
		}
	}
	return n
}

// SlotJitterMS estimates slot timing jitter from recent stream freshness.
func (s *SlotState) SlotJitterMS() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastUpdate.IsZero() {
		return 0
	}
	drift := time.Since(s.lastUpdate).Milliseconds()
	if drift > 400 {
		return float64(drift) / 10.0
	}
	return 2.4
}

// ValidatorStabilityPct estimates leader stability from skipped slots in the window.
func (s *SlotState) ValidatorStabilityPct() int {
	skipped := s.SkippedSlotsInWindow()
	stability := 100 - skipped*3
	if stability < 50 {
		return 50
	}
	if stability > 99 {
		return 99
	}
	return stability
}

func (s *SlotState) Snapshot() (uint64, float64, []SlotEntry, map[uint64]string, time.Time, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	leadersCopy := make(map[uint64]string, len(s.leaders))
	for k, v := range s.leaders {
		leadersCopy[k] = v
	}
	slotsCopy := make([]SlotEntry, len(s.recentSlots))
	copy(slotsCopy, s.recentSlots)
	now := time.Now()
	for i := range slotsCopy {
		slotsCopy[i].AgeMS = now.Sub(slotsCopy[i].SeenAt).Milliseconds()
	}
	return s.currentSlot, s.tps, slotsCopy, leadersCopy, s.lastUpdate, s.streamLagSlots
}

func (s *SlotState) IncReconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnectCount++
}

func isJitoLeader(leader string) bool {
	jitoNames := []string{"Jito", "jito", "JITO"}
	for _, n := range jitoNames {
		if leader != "" && (leader == n || contains(leader, n)) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
