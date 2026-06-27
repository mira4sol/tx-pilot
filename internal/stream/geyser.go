package stream

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Subscriber struct {
	client pb.GeyserClient
	logger *zap.Logger
}

func NewSubscriber(conn *grpc.ClientConn, logger *zap.Logger) *Subscriber {
	return &Subscriber{client: pb.NewGeyserClient(conn), logger: logger}
}

func (s *Subscriber) Run(ctx context.Context, authCtx func(context.Context) context.Context, slotState *SlotState, signatures chan<- string) error {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := s.subscribeOnce(ctx, authCtx, slotState, signatures)
		if err != nil {
			slotState.IncReconnect()
			s.logger.Warn("geyser stream disconnected, reconnecting", zap.Error(err), zap.Duration("backoff", backoff))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (s *Subscriber) subscribeOnce(ctx context.Context, authCtx func(context.Context) context.Context, slotState *SlotState, signatures chan<- string) error {
	stream, err := s.client.Subscribe(authCtx(ctx))
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	req := &pb.SubscribeRequest{
		Slots: map[string]*pb.SubscribeRequestFilterSlots{"slots": {}},
		Transactions: map[string]*pb.SubscribeRequestFilterTransactions{
			"txs": {Vote: boolPtr(false), Failed: boolPtr(false)},
		},
	}
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("send subscribe request: %w", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("stream closed")
			}
			return err
		}
		if slot := update.GetSlot(); slot != nil {
			slotState.UpdateSlot(slot.Slot)
		}
		if tx := update.GetTransaction(); tx != nil && tx.Transaction != nil {
			sig := fmt.Sprintf("%x", tx.Transaction.Signature)
			select {
			case signatures <- sig:
			default:
			}
		}
	}
}

func boolPtr(v bool) *bool { return &v }
