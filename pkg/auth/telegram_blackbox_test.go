package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTelegramUser_JSON(t *testing.T) {
	tests := []struct {
		name     string
		user     TelegramUser
		expected string
	}{
		{
			name: "complete user",
			user: TelegramUser{
				ID:        123456789,
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
				PhotoURL:  "https://t.me/i/userpic/320/johndoe.jpg",
				IsBot:     false,
			},
			expected: `{"id":123456789,"first_name":"John","last_name":"Doe","username":"johndoe","photo_url":"https://t.me/i/userpic/320/johndoe.jpg"}`,
		},
		{
			name: "minimal user",
			user: TelegramUser{
				ID:        987654321,
				FirstName: "Jane",
			},
			expected: `{"id":987654321,"first_name":"Jane"}`,
		},
		{
			name: "bot user",
			user: TelegramUser{
				ID:        555666777,
				FirstName: "TestBot",
				IsBot:     true,
			},
			expected: `{"id":555666777,"first_name":"TestBot","is_bot":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.user)
			if err != nil {
				t.Fatalf("Failed to marshal user: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}

			var unmarshaled TelegramUser
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal user: %v", err)
			}

			if unmarshaled.ID != tt.user.ID {
				t.Errorf("Expected ID %d, got %d", tt.user.ID, unmarshaled.ID)
			}
			if unmarshaled.FirstName != tt.user.FirstName {
				t.Errorf("Expected FirstName %s, got %s", tt.user.FirstName, unmarshaled.FirstName)
			}
			if unmarshaled.IsBot != tt.user.IsBot {
				t.Errorf("Expected IsBot %v, got %v", tt.user.IsBot, unmarshaled.IsBot)
			}
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectUser     bool
		expectedUserID int64
	}{
		{
			name: "user in context",
			setupContext: func() context.Context {
				user := &TelegramUser{
					ID:        123456789,
					FirstName: "Test",
					Username:  "testuser",
				}
				return context.WithValue(context.Background(), userKey, user)
			},
			expectUser:     true,
			expectedUserID: 123456789,
		},
		{
			name: "empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectUser: false,
		},
		{
			name: "wrong type in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), userKey, "not a user")
			},
			expectUser: false,
		},
		{
			name: "nil user in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), userKey, (*TelegramUser)(nil))
			},
			expectUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			user, ok := GetUserFromContext(ctx)

			if tt.expectUser {
				if !ok {
					t.Fatal("Expected to find user in context")
				}
				if user != nil && user.ID != tt.expectedUserID {
					t.Errorf("Expected user ID %d, got %d", tt.expectedUserID, user.ID)
				}
			} else {
				if ok {
					t.Error("Expected no user in context")
				}
			}
		})
	}
}

func TestTelegramAuthMiddleware_ValidRequest(t *testing.T) {
	user := TelegramUser{
		ID:        123456789,
		FirstName: "Test",
		Username:  "testuser",
		IsBot:     false,
	}

	params := createValidAuthParams(t, user, "test_bot_token")

	var capturedUser *TelegramUser
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, ok := GetUserFromContext(r.Context()); ok {
			capturedUser = user
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := TelegramAuthMiddleware("test_bot_token")
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "tma "+params.Encode())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedUser == nil {
		t.Fatal("Expected user to be captured in context")
	}

	if capturedUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, capturedUser.ID)
	}

	if capturedUser.FirstName != user.FirstName {
		t.Errorf("Expected FirstName %s, got %s", user.FirstName, capturedUser.FirstName)
	}
}

func TestTelegramAuthMiddleware_InvalidSignature(t *testing.T) {
	params := url.Values{}
	params.Set("hash", "invalid_hash")
	params.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	params.Set("user", `{"id":123456789,"first_name":"Test"}`)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TelegramAuthMiddleware("test_bot_token")
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "tma "+params.Encode())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTelegramAuthMiddleware_BotUser(t *testing.T) {
	botUser := TelegramUser{
		ID:        987654321,
		FirstName: "TestBot",
		IsBot:     true,
	}

	params := createValidAuthParams(t, botUser, "test_bot_token")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TelegramAuthMiddleware("test_bot_token")
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "tma "+params.Encode())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestTelegramAuthMiddleware_MissingParameters(t *testing.T) {
	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
	}{
		{
			name: "missing authorization header",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid authorization format",
			setupAuth: func() string {
				return "invalid"
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong auth type",
			setupAuth: func() string {
				return "Bearer token123"
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing hash",
			setupAuth: func() string {
				params := url.Values{
					"auth_date": []string{strconv.FormatInt(time.Now().Unix(), 10)},
					"user":      []string{`{"id":123456789,"first_name":"Test"}`},
				}
				return "tma " + params.Encode()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing auth_date",
			setupAuth: func() string {
				params := url.Values{
					"hash": []string{"abc123"},
					"user": []string{`{"id":123456789,"first_name":"Test"}`},
				}
				return "tma " + params.Encode()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing user",
			setupAuth: func() string {
				params := url.Values{
					"hash":      []string{"abc123"},
					"auth_date": []string{strconv.FormatInt(time.Now().Unix(), 10)},
				}
				return "tma " + params.Encode()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "auth_date in future",
			setupAuth: func() string {
				params := url.Values{
					"hash":      []string{"invalid_hash"},
					"auth_date": []string{strconv.FormatInt(time.Now().Add(2*time.Minute).Unix(), 10)},
					"user":      []string{`{"id":123456789,"first_name":"Test"}`},
				}
				return "tma " + params.Encode()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "auth_date too old",
			setupAuth: func() string {
				params := url.Values{
					"hash":      []string{"invalid_hash"},
					"auth_date": []string{strconv.FormatInt(time.Now().Add(-2*time.Minute).Unix(), 10)},
					"user":      []string{`{"id":123456789,"first_name":"Test"}`},
				}
				return "tma " + params.Encode()
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := TelegramAuthMiddleware("test_bot_token")
			handler := middleware(testHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			authHeader := tt.setupAuth()
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func createValidAuthParams(t *testing.T, user TelegramUser, botToken string) url.Values {
	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	params := url.Values{}
	params.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	params.Set("user", string(userJSON))
	params.Set("query_id", "test_query_id")

	dataCheckString := buildDataCheckString(params)
	hash := generateValidHashWithLib(dataCheckString, botToken)
	params.Set("hash", hash)

	return params
}

func generateValidHashWithLib(dataCheckString, botToken string) string {
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))
	secret := secretKey.Sum(nil)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil))
}

func buildDataCheckString(values url.Values) string {
	var parts []string
	for key, vals := range values {
		if key == "hash" {
			continue
		}
		if len(vals) > 0 {
			parts = append(parts, key+"="+vals[0])
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}
