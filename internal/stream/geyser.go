package stream

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const maxTrackedTxFilters = 64

// SignatureHandler is invoked when a tracked signature is observed on the Geyser stream.
type SignatureHandler func(sig string, slot uint64)

type Subscriber struct {
	client    pb.GeyserClient
	logger    *zap.Logger
	tracker   *SignatureTracker
	onSig     SignatureHandler
	refreshCh chan struct{}
}

func NewSubscriber(conn *grpc.ClientConn, logger *zap.Logger, tracker *SignatureTracker) *Subscriber {
	s := &Subscriber{
		client:    pb.NewGeyserClient(conn),
		logger:    logger,
		tracker:   tracker,
		refreshCh: make(chan struct{}, 1),
	}
	if tracker != nil {
		tracker.SetOnChange(s.scheduleRefresh)
	}
	return s
}

func (s *Subscriber) SetHandler(fn SignatureHandler) {
	s.onSig = fn
}

func (s *Subscriber) scheduleRefresh() {
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}
}

// Run maintains a resilient Geyser subscription, reconnecting with exponential backoff
// until ctx is cancelled. The server process does not need to restart on disconnect.
func (s *Subscriber) Run(ctx context.Context, authCtx func(context.Context) context.Context, slotState *SlotState) error {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := s.streamSession(ctx, authCtx, slotState)
		if err == nil {
			backoff = time.Second
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slotState.IncReconnect()
		s.logger.Warn("geyser stream disconnected, reconnecting",
			zap.Error(err), zap.Duration("backoff", backoff))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (s *Subscriber) streamSession(ctx context.Context, authCtx func(context.Context) context.Context, slotState *SlotState) error {
	stream, err := s.client.Subscribe(authCtx(ctx))
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	if err := s.sendSubscribe(stream); err != nil {
		return fmt.Errorf("send subscribe request: %w", err)
	}
	s.logger.Info("geyser stream connected",
		zap.Int("tracked_tx_filters", s.trackedFilterCount()))

	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var refreshWG sync.WaitGroup
	refreshWG.Add(1)
	go func() {
		defer refreshWG.Done()
		s.runRefreshLoop(sessionCtx, stream)
	}()

	recvErr := s.recvLoop(sessionCtx, stream, slotState)
	cancel()
	refreshWG.Wait()
	return recvErr
}

func (s *Subscriber) runRefreshLoop(ctx context.Context, stream pb.Geyser_SubscribeClient) {
	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.refreshCh:
			debounce.Reset(150 * time.Millisecond)
		case <-debounce.C:
			if err := s.sendSubscribe(stream); err != nil {
				s.logger.Warn("geyser subscription refresh failed", zap.Error(err))
				return
			}
			s.logger.Debug("geyser transaction filters refreshed",
				zap.Int("tracked_tx_filters", s.trackedFilterCount()))
		}
	}
}

func (s *Subscriber) recvLoop(ctx context.Context, stream pb.Geyser_SubscribeClient, slotState *SlotState) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		update, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("stream closed")
			}
			return err
		}
		if slot := update.GetSlot(); slot != nil {
			slotState.UpdateSlot(slot.Slot)
			continue
		}
		tx := update.GetTransaction()
		if tx == nil || tx.Transaction == nil {
			continue
		}
		sig := solana.SignatureFromBytes(tx.Transaction.Signature).String()
		if _, ok := s.tracker.Lookup(sig); !ok {
			continue
		}
		slot := tx.GetSlot()
		if slot == 0 {
			slot = slotState.CurrentSlot()
		}
		if s.onSig != nil {
			go s.onSig(sig, slot)
		}
	}
}

func (s *Subscriber) sendSubscribe(stream pb.Geyser_SubscribeClient) error {
	return stream.Send(s.buildSubscribeRequest())
}

func (s *Subscriber) buildSubscribeRequest() *pb.SubscribeRequest {
	req := &pb.SubscribeRequest{
		Slots: map[string]*pb.SubscribeRequestFilterSlots{
			"slots": {},
		},
	}
	if s.tracker == nil {
		return req
	}
	sigs := s.tracker.Signatures(maxTrackedTxFilters)
	if len(sigs) == 0 {
		return req
	}
	txs := make(map[string]*pb.SubscribeRequestFilterTransactions, len(sigs))
	for i, sig := range sigs {
		sigCopy := sig
		txs[fmt.Sprintf("tracked_%d", i)] = &pb.SubscribeRequestFilterTransactions{
			Vote:      boolPtr(false),
			Failed:    boolPtr(true),
			Signature: &sigCopy,
		}
	}
	req.Transactions = txs
	return req
}

func (s *Subscriber) trackedFilterCount() int {
	if s.tracker == nil {
		return 0
	}
	return len(s.tracker.Signatures(maxTrackedTxFilters))
}

func boolPtr(v bool) *bool { return &v }
