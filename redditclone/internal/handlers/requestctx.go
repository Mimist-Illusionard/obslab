package handlers

import (
	"context"

	"go.uber.org/zap"
)

func Logger(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value("logger").(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return logger
}
