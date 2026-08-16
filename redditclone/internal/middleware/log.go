package middleware

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/Mimist-Illusionard/obslab/internal/metrics"
	"go.uber.org/zap"
)

const (
	idLength        = 10
	requestIDHeader = "X-Request-ID"
	requestID       = "request_id"
	loggerKey       = "logger"
	metricsKey      = "metrics"
	charset         = "abcdefghijklmnopqrstuvwxyz0123456789_-"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type Logger struct {
	length  int
	key     string
	charset string
	zap     *zap.Logger
	metrics *metrics.Metrics
}

func NewLogger(length int, key, charset string, logger *zap.Logger, metrics *metrics.Metrics) *Logger {
	return &Logger{length: length, key: key, charset: charset, zap: logger, metrics: metrics}
}

func (logger *Logger) LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		reqID := generateID()

		requestLogger := logger.zap.With(
			zap.String("request_id", reqID),
		)

		ctx := r.Context()
		ctx = context.WithValue(ctx, requestID, reqID)
		ctx = context.WithValue(ctx, loggerKey, requestLogger)
		ctx = context.WithValue(ctx, metricsKey, logger.metrics)

		r = r.WithContext(ctx)

		rw := newResponseWriter(w)

		rw.Header().Add(requestIDHeader, reqID)

		next.ServeHTTP(rw, r)

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", time.Since(start)),
		}

		if r.URL.Path != "/metrics" {
			logger.metrics.Hits.WithLabelValues(strconv.Itoa(rw.statusCode), r.URL.Path).Inc()
		}

		switch {
		case rw.statusCode >= 500:
			requestLogger.Error("req failed", fields...)
		case rw.statusCode >= 400:
			requestLogger.Warn("req warn", fields...)
		default:
			requestLogger.Info("req completed", fields...)
		}
	})
}

func generateID() string {
	b := make([]byte, idLength)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}
