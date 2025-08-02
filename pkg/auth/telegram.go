package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	initdata "github.com/telegram-mini-apps/init-data-golang"
)

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}

type ctxKey string

const (
	userKey     ctxKey        = "telegram_user"
	authTimeout time.Duration = 24 * time.Hour
)

func GetUserFromContext(ctx context.Context) (*TelegramUser, bool) {
	u, ok := ctx.Value(userKey).(*TelegramUser)
	return u, ok
}

func TelegramAuthMiddleware(botToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			authParts := strings.Split(authHeader, " ")
			if len(authParts) != 2 {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			authType := authParts[0]
			authData := authParts[1]

			if authType != "tma" {
				http.Error(w, "Invalid authorization type", http.StatusUnauthorized)
				return
			}

			if err := initdata.Validate(authData, botToken, authTimeout); err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			parsedData, err := initdata.Parse(authData)
			if err != nil {
				http.Error(w, "Invalid init data format", http.StatusUnauthorized)
				return
			}

			if parsedData.User.ID == 0 {
				http.Error(w, "User data not found", http.StatusUnauthorized)
				return
			}

			user := TelegramUser{
				ID:        parsedData.User.ID,
				FirstName: parsedData.User.FirstName,
				LastName:  parsedData.User.LastName,
				Username:  parsedData.User.Username,
				PhotoURL:  parsedData.User.PhotoURL,
				IsBot:     parsedData.User.IsBot,
			}

			if user.IsBot {
				http.Error(w, "Forbidden: bots are not allowed", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
