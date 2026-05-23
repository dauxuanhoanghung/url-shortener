package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenExpiry       = 15 * time.Minute
	RefreshTokenExpiry      = 30 * 24 * time.Hour
	AdminAccessTokenExpiry  = 5 * time.Minute
	AdminRefreshTokenExpiry = 8 * time.Hour
)

type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenInput carries the per-user claims needed to mint a token. Using a struct
// (instead of positional args) means future fields can be added without
// touching every caller.
type TokenInput struct {
	UserID string
	Email  string
	Role   string
}

func GenerateAccessToken(in TokenInput, secret string) (string, error) {
	return signToken(in, accessExpiryFor(in.Role), secret)
}

func GenerateRefreshToken(in TokenInput, secret string) (string, error) {
	return signToken(in, refreshExpiryFor(in.Role), secret)
}

func ValidateToken(tokenString, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

func signToken(in TokenInput, expiry time.Duration, secret string) (string, error) {
	claims := TokenClaims{
		UserID: in.UserID,
		Email:  in.Email,
		Role:   in.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func accessExpiryFor(role string) time.Duration {
	if role == "admin" {
		return AdminAccessTokenExpiry
	}
	return AccessTokenExpiry
}

func refreshExpiryFor(role string) time.Duration {
	if role == "admin" {
		return AdminRefreshTokenExpiry
	}
	return RefreshTokenExpiry
}
