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

// LogDBOp logs Error on failure or Info on success for a database operation.
func LogDBOp(logger *zap.Logger, op string, err error, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if err != nil {
		all := append([]zap.Field{zap.String("op", op), zap.Error(err)}, fields...)
		logger.Error("database operation failed", all...)
		return
	}
	all := append([]zap.Field{zap.String("op", op)}, fields...)
	logger.Info("database operation committed", all...)
}

// LogDBResult logs the outcome of a write scoped to a transaction id.
func LogDBResult(logger *zap.Logger, op, txID string, err error) {
	LogDBOp(logger, op, err, zap.String("transaction_id", txID))
}

// MustContext returns ctx or background when nil.
func MustContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
