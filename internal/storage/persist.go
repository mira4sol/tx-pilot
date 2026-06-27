package storage

import (
	"context"

	"go.uber.org/zap"
)

// LogDB logs a database operation error without failing the caller's control flow.
func LogDB(logger *zap.Logger, op string, err error) {
	if err == nil || logger == nil {
		return
	}
	logger.Error("database operation failed", zap.String("op", op), zap.Error(err))
}

// MustContext returns ctx or background when nil.
func MustContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
