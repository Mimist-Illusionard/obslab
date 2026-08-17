package logs

import (
	"context"

	sentryzap "github.com/getsentry/sentry-go/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerType string

const (
	Zap    LoggerType = "Zap"
	Sentry LoggerType = "Sentry"
)

func NewLogger(loggerType LoggerType) (*zap.Logger, error) {
	switch loggerType {
	case Zap:
		zapLogger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}

		return zapLogger, nil
	case Sentry:

		ctx := context.Background()
		sentryCore := sentryzap.NewSentryCore(ctx, sentryzap.Option{
			Level: []zapcore.Level{
				zapcore.InfoLevel,
				zapcore.WarnLevel,
				zapcore.ErrorLevel,
			},
			AddCaller: true,
		})

		logger := zap.New(sentryCore)
		return logger, nil
	}

	return nil, nil
}
