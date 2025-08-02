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
	SecretKey     string
	TokenDuration time.Duration
	Issuer        string
}

func DefaultJWTConfig(secretKey string) *JWTConfig {
	return &JWTConfig{
		SecretKey:     secretKey,
		TokenDuration: 24 * time.Hour,
		Issuer:        "clientpulse-org",
	}
}

type JWTCustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}

type jwtCtxKey string

const (
	jwtUserKey jwtCtxKey = "jwt_user"
)

func IssueJWTFromTelegramUser(user *TelegramUser, config *JWTConfig) (string, error) {
	if user == nil {
		return "", errors.New("user cannot be nil")
	}

	if config.SecretKey == "" {
		return "", errors.New("secret key cannot be empty")
	}

	now := time.Now()

	claims := JWTCustomClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"clientpulse-org"},
			ExpiresAt: jwt.NewNumericDate(now.Add(config.TokenDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

func JWTAuthMiddleware(config *JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.ParseWithClaims(tokenString, &JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(config.SecretKey), nil
			})

			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(*JWTCustomClaims)
			if !ok || !token.Valid {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), jwtUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetJWTUserFromContext(ctx context.Context) (*JWTCustomClaims, bool) {
	claims, ok := ctx.Value(jwtUserKey).(*JWTCustomClaims)
	return claims, ok
}

func RefreshJWTToken(tokenString string, config *JWTConfig) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTCustomClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	now := time.Now()
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(config.TokenDuration))
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)
	claims.RegisteredClaims.ID = generateTokenID()

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return newToken.SignedString([]byte(config.SecretKey))
}

func ValidateJWTToken(tokenString string, config *JWTConfig) (*JWTCustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func generateTokenID() string {
	b := make([]byte, 16) // increase to 32 for more entropy
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func JWTOptionalMiddleware(config *JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := ValidateJWTToken(tokenString, config)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), jwtUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
