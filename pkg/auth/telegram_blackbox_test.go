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

			// Test unmarshaling back
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
				// Use the actual context key from the package
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
			expectUser: true, // Go returns true for nil pointer in type assertion
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
	// Test with valid auth data
	user := TelegramUser{
		ID:        123456789,
		FirstName: "Test",
		Username:  "testuser",
		IsBot:     false,
	}

	// Create valid auth parameters
	params := createValidAuthParams(t, user, "test_bot_token")

	// Create test handler
	var capturedUser *TelegramUser
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, ok := GetUserFromContext(r.Context()); ok {
			capturedUser = user
		}
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware
	middleware := TelegramAuthMiddleware("test_bot_token")
	handler := middleware(testHandler)

	// Create request
	req := httptest.NewRequest("GET", "/test?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify response
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
	// Test with invalid signature
	params := url.Values{}
	params.Set("hash", "invalid_hash")
	params.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	params.Set("user", `{"id":123456789,"first_name":"Test"}`)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TelegramAuthMiddleware("test_bot_token")
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTelegramAuthMiddleware_BotUser(t *testing.T) {
	// Test with bot user (should be rejected)
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

	req := httptest.NewRequest("GET", "/test?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestTelegramAuthMiddleware_MissingParameters(t *testing.T) {
	tests := []struct {
		name           string
		params         url.Values
		expectedStatus int
	}{
		{
			name: "missing hash",
			params: url.Values{
				"auth_date": []string{strconv.FormatInt(time.Now().Unix(), 10)},
				"user":      []string{`{"id":123456789,"first_name":"Test"}`},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing auth_date",
			params: url.Values{
				"hash": []string{"abc123"},
				"user": []string{`{"id":123456789,"first_name":"Test"}`},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing user",
			params: url.Values{
				"hash":      []string{"abc123"},
				"auth_date": []string{strconv.FormatInt(time.Now().Unix(), 10)},
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

			req := httptest.NewRequest("GET", "/test?"+tt.params.Encode(), nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// Helper function to create valid auth parameters for testing
func createValidAuthParams(t *testing.T, user TelegramUser, botToken string) url.Values {
	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	params := url.Values{}
	params.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	params.Set("user", string(userJSON))

	// Generate valid hash using the same logic as the actual implementation
	hash := generateValidHash(params)
	params.Set("hash", hash)

	return params
}

// Helper function to generate valid hash for testing
func generateValidHash(values url.Values) string {
	// Use the actual HMAC-SHA256 logic from the package
	dataCheck := buildDataCheckString(values)
	secret := sha256.Sum256([]byte("test_bot_token"))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheck))
	return hex.EncodeToString(mac.Sum(nil))
}

// Helper function to build data check string
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
	// Sort parts for consistent ordering
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}
