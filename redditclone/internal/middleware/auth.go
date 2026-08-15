package middleware

import (
	"context"
	"fmt"
	"net/http"

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
		token := r.Header.Get("Authorization")
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		token = token[7:] //We know that in header will be Bearer token

		claims, err := a.jwtS.ParseJwt(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			a.logger.Error(
				fmt.Sprintf("jwt parse error: %v", err),
				zap.String("request_id", w.Header().Get("X-Request-ID")))
			return
		}

		ctx := context.WithValue(r.Context(), "claims", *claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
