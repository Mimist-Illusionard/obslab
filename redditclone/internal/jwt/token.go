package jwt

import (
	"fmt"
	"time"

	"github.com/Mimist-Illusionard/obslab/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	User models.User `json:"user"`
	jwt.RegisteredClaims
}

type Token struct {
	Token string `json:"token"`
}

type JwtService struct {
	secret string
}

func NewJwtGenerator(secret string) *JwtService {
	return &JwtService{secret: secret}
}

func (jwtGen *JwtService) GenerateJwt(user *models.User) (*Token, error) {
	claims := Claims{
		User: *user,
	}

	claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(168 * time.Hour)) //7 days

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtGen.secret))
	if err != nil {
		return nil, err
	}

	return &Token{Token: tokenString}, nil
}

func (jwtGen *JwtService) ParseJwt(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("bad sign method")
		}
		return []byte(jwtGen.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("bad token: %v", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid: %v", err)
	}

	return claims, nil
}
