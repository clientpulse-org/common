package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueJWTFromTelegramUser(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	user := &TelegramUser{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		PhotoURL:  "https://example.com/photo.jpg",
		IsBot:     false,
	}

	token, err := IssueJWTFromTelegramUser(user, config)
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	if token == "" {
		t.Fatal("Token should not be empty")
	}

	claims, err := ValidateJWTToken(token, config)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}

	if claims.Username != user.Username {
		t.Errorf("Expected Username %s, got %s", user.Username, claims.Username)
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	user := &TelegramUser{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		PhotoURL:  "https://example.com/photo.jpg",
		IsBot:     false,
	}

	token, err := IssueJWTFromTelegramUser(user, config)
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetJWTUserFromContext(r.Context())
		if !ok {
			t.Error("User claims not found in context")
			return
		}

		if claims.UserID != user.ID {
			t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
		}

		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTAuthMiddleware(config)
	server := httptest.NewServer(middleware(handler))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuthMiddleware_NoToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when no token is provided")
	})

	middleware := JWTAuthMiddleware(config)
	server := httptest.NewServer(middleware(handler))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when invalid token is provided")
	})

	middleware := JWTAuthMiddleware(config)
	server := httptest.NewServer(middleware(handler))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer invalid-token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestJWTOptionalMiddleware(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	user := &TelegramUser{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		PhotoURL:  "https://example.com/photo.jpg",
		IsBot:     false,
	}

	token, err := IssueJWTFromTelegramUser(user, config)
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetJWTUserFromContext(r.Context())
		if !ok {
			t.Error("User claims not found in context")
			return
		}

		if claims.UserID != user.ID {
			t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
		}

		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTOptionalMiddleware(config)
	server := httptest.NewServer(middleware(handler))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestJWTOptionalMiddleware_NoToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, ok := GetJWTUserFromContext(r.Context())
		if ok {
			t.Error("User claims should not be present when no token is provided")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTOptionalMiddleware(config)
	server := httptest.NewServer(middleware(handler))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !handlerCalled {
		t.Error("Handler should be called even when no token is provided")
	}
}

func TestRefreshJWTToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	config.TokenDuration = 1 * time.Hour

	user := &TelegramUser{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		PhotoURL:  "https://example.com/photo.jpg",
		IsBot:     false,
	}

	originalToken, err := IssueJWTFromTelegramUser(user, config)
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	refreshedToken, err := RefreshJWTToken(originalToken, config)
	if err != nil {
		t.Fatalf("Failed to refresh JWT: %v", err)
	}

	if refreshedToken == originalToken {
		t.Error("Refreshed token should be different from original token")
	}

	claims, err := ValidateJWTToken(refreshedToken, config)
	if err != nil {
		t.Fatalf("Failed to validate refreshed JWT: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}

}

func TestGetJWTUserFromContext(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")

	user := &TelegramUser{
		ID:        123456789,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		PhotoURL:  "https://example.com/photo.jpg",
		IsBot:     false,
	}

	token, err := IssueJWTFromTelegramUser(user, config)
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	claims, err := ValidateJWTToken(token, config)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	ctx := context.WithValue(context.Background(), jwtUserKey, claims)

	retrievedClaims, ok := GetJWTUserFromContext(ctx)
	if !ok {
		t.Fatal("Failed to retrieve claims from context")
	}

	if retrievedClaims.UserID != claims.UserID {
		t.Errorf("Expected UserID %d, got %d", claims.UserID, retrievedClaims.UserID)
	}

	emptyCtx := context.Background()
	_, ok = GetJWTUserFromContext(emptyCtx)
	if ok {
		t.Error("Should not find claims in empty context")
	}
}
