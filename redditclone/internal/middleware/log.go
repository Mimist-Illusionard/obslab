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
	charset         = "abcdefghijklmnopqrstuvwxyz0123456789"
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

		rw := newResponseWriter(w)
		rw.Header().Add(requestIDHeader, reqID)
		ctx := context.WithValue(r.Context(), "metrics", logger.metrics)

		next.ServeHTTP(rw, r.WithContext(ctx))

		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", r.Method),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("url", r.URL.Path),
			zap.Int("status", rw.statusCode),
			zap.Duration("work_time", time.Since(start)),
		}

		if r.URL.Path != "/metrics" {
			logger.metrics.Hits.WithLabelValues(strconv.Itoa(rw.statusCode), r.URL.Path).Inc()
		}

		if rw.statusCode >= 500 {
			logger.zap.Error("request failed", fields...)
			return
		}

		if rw.statusCode >= 400 {
			logger.zap.Warn("request completed with client error", fields...)
			return
		}

		logger.zap.Info("request completed", fields...)
	})
}

func generateID() string {
	b := make([]byte, idLength)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}
