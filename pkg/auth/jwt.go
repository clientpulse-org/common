// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Issuer    string
	Audience  string
	AccessTTL time.Duration
	SecretKey []byte // HS256 key
}

type UserIdentity struct {
	UserID string
}

type jwtCtxKey string

const (
	TokenLength = 16

	jwtUserKey jwtCtxKey = "user_id"
)

func IssueAccessJWT(user UserIdentity, cfg *JWTConfig) (string, error) {
	if len(cfg.SecretKey) == 0 {
		return "", errors.New("secret key cannot be empty")
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   user.UserID,
		Issuer:    cfg.Issuer,
		Audience:  []string{cfg.Audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        generateTokenID(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.SecretKey)
}

func ValidateAccessJWT(tokenString string, cfg *JWTConfig) (userID string, err error) {
	if len(cfg.SecretKey) == 0 {
		return "", errors.New("secret key cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cfg.SecretKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	return claims.Subject, nil
}

func RequireAuth(cfg *JWTConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := ValidateAccessJWT(tokenString, cfg)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), jwtUserKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(jwtUserKey).(string)
	return userID, ok
}

func generateTokenID() string {
	b := make([]byte, TokenLength)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
