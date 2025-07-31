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

// JWTConfig holds configuration for JWT operations
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
	Issuer        string
}

// DefaultJWTConfig returns a default JWT configuration
func DefaultJWTConfig(secretKey string) *JWTConfig {
	return &JWTConfig{
		SecretKey:     secretKey,
		TokenDuration: 24 * time.Hour, // 24 hours default
		Issuer:        "clientpulse-org",
	}
}

// JWTCustomClaims represents the custom claims for JWT tokens
type JWTCustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}

// ctxKey is a private type used to avoid key collisions in context.
type jwtCtxKey string

const (
	// jwtUserKey is the context key under which the JWT user claims are stored.
	jwtUserKey jwtCtxKey = "jwt_user"
)

// IssueJWTFromTelegramUser creates a JWT token from Telegram user data
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

// JWTAuthMiddleware returns an HTTP middleware that:
// 1. Extracts JWT token from Authorization header
// 2. Validates the token signature and claims
// 3. Injects the user claims into the request context
func JWTAuthMiddleware(config *JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check if it's a Bearer token
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the token
			token, err := jwt.ParseWithClaims(tokenString, &JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(config.SecretKey), nil
			})

			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Extract claims
			claims, ok := token.Claims.(*JWTCustomClaims)
			if !ok || !token.Valid {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Store the user claims in the request context
			ctx := context.WithValue(r.Context(), jwtUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetJWTUserFromContext retrieves the JWT user claims from ctx.
// It returns the claims pointer and a boolean indicating presence.
func GetJWTUserFromContext(ctx context.Context) (*JWTCustomClaims, bool) {
	claims, ok := ctx.Value(jwtUserKey).(*JWTCustomClaims)
	return claims, ok
}

// RefreshJWTToken creates a new JWT token with extended expiration
func RefreshJWTToken(tokenString string, config *JWTConfig) (string, error) {
	// Parse the existing token without validation to get claims
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTCustomClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	// Create new token with extended expiration
	now := time.Now()
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(config.TokenDuration))
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)
	claims.RegisteredClaims.ID = generateTokenID()

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return newToken.SignedString([]byte(config.SecretKey))
}

// ValidateJWTToken validates a JWT token and returns the claims
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

// generateTokenID generates a random token ID
func generateTokenID() string {
	b := make([]byte, 16) // increase to 32 for more entropy
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// JWTOptionalMiddleware returns an HTTP middleware that:
// 1. Optionally extracts and validates JWT token from Authorization header
// 2. If valid token is present, injects the user claims into the request context
// 3. If no token or invalid token, continues without user context (doesn't return error)
func JWTOptionalMiddleware(config *JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				// No token provided, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Try to parse and validate the token
			claims, err := ValidateJWTToken(tokenString, config)
			if err != nil {
				// Invalid token, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Valid token, store the user claims in the request context
			ctx := context.WithValue(r.Context(), jwtUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
