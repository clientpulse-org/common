package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TelegramUser represents the authenticated user information
// provided by Telegram WebApp (WebAppData.user).
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}

// ctxKey is a private type used to avoid key collisions in context.
type ctxKey string

const (
	// userKey is the context key under which the TelegramUser is stored.
	userKey ctxKey = "telegram_user"

	// authTimeout defines how long auth_data remains valid.
	authTimeout = 24 * time.Hour
)

// GetUserFromContext retrieves the TelegramUser from ctx.
// It returns the user pointer and a boolean indicating presence.
func GetUserFromContext(ctx context.Context) (*TelegramUser, bool) {
	u, ok := ctx.Value(userKey).(*TelegramUser)
	return u, ok
}

// TelegramAuthMiddleware returns an HTTP middleware that:
// 1. Verifies the Telegram WebApp auth_data signature and timestamp.
// 2. Parses the user JSON payload.
// 3. Rejects bot accounts.
// 4. Injects the TelegramUser into the request context.
func TelegramAuthMiddleware(botToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			// Validate signature and timestamp of auth_data parameters.
			if err := validateAuth(query, botToken); err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Extract and unmarshal the user JSON payload.
			userJSON := query.Get("user")
			var user TelegramUser
			if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
				http.Error(w, "Unauthorized: invalid user data", http.StatusUnauthorized)
				return
			}

			// Deny access for bot users.
			if user.IsBot {
				http.Error(w, "Forbidden: bots are not allowed", http.StatusForbidden)
				return
			}

			// Store the authenticated user in the request context.
			ctx := context.WithValue(r.Context(), userKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateAuth checks that auth_data contains a valid HMAC-SHA256 signature
// generated with the bot token, and that auth_date is recent.
func validateAuth(values url.Values, botToken string) error {
	// Ensure required parameters are present.
	hash := values.Get("hash")
	if hash == "" {
		return errors.New("missing hash parameter")
	}
	ts := values.Get("auth_date")
	if ts == "" {
		return errors.New("missing auth_date parameter")
	}

	// Parse auth_date as Unix timestamp and check freshness.
	seconds, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("invalid auth_date format")
	}
	authTime := time.Unix(seconds, 0)
	timeDiff := time.Since(authTime)

	// Check if auth_date is too far in the past OR in the future
	if timeDiff > authTimeout || timeDiff < -authTimeout {
		return errors.New("authorization data expired")
	}

	// Build the data check string by concatenating all query params (except hash).
	var parts []string
	for key, vals := range values {
		if key == "hash" {
			continue
		}
		parts = append(parts, key+"="+vals[0])
	}
	sort.Strings(parts)
	dataCheck := strings.Join(parts, "\n")

	// Compute expected HMAC-SHA256 value using the bot token as key.
	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheck))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// Compare the expected signature with the provided hash.
	if !hmac.Equal([]byte(expectedMAC), []byte(hash)) {
		return errors.New("signature mismatch")
	}
	return nil
}
