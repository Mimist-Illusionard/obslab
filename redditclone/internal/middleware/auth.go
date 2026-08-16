package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"go.uber.org/zap"
)

type Auth struct {
	jwtS   *jwt.JwtService
	logger *zap.Logger
}

func NewAuth(s *jwt.JwtService, zap *zap.Logger) *Auth {
	return &Auth{jwtS: s, logger: zap}
}

func (a *Auth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger, ok := r.Context().Value("logger").(*zap.Logger)
		if !ok {
			logger = zap.NewNop()
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(token, "Bearer ") {
			logger.Warn("invalid authorization header")

			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := a.jwtS.ParseJwt(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			req := w.Header().Get("X-Request-ID")
			a.logger.Error(
				fmt.Sprintf("jwt parse error: %v", err),
				zap.String("request_id", req))
			return
		}

		ctx := context.WithValue(r.Context(), "claims", *claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
