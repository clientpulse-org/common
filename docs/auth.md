# JWT Authentication Package

This package provides JWT (JSON Web Token) authentication functionality that integrates seamlessly with Telegram user authentication. It allows you to issue JWT tokens from Telegram user data and protect your API endpoints with JWT middleware.

## Features

- **JWT Token Issuance**: Create JWT tokens from Telegram user data
- **JWT Middleware**: Protect HTTP endpoints with JWT authentication
- **Optional JWT Middleware**: Allow endpoints to work with or without JWT tokens
- **Token Validation**: Validate JWT tokens and extract user claims
- **Token Refresh**: Refresh JWT tokens with extended expiration
- **Context Integration**: Access user claims from HTTP request context

## Installation

The package uses the `github.com/golang-jwt/jwt/v5` library for JWT operations. Make sure it's included in your `go.mod`:

```go
require github.com/golang-jwt/jwt/v5 v5.2.0
```

## Quick Start

### 1. Basic JWT Token Issuance

```go
package main

import (
    "fmt"
    "time"
    "github.com/quiby-ai/common/pkg/auth"
)

func main() {
    // Configure JWT settings
    jwtConfig := auth.DefaultJWTConfig("your-secret-key-here")
    jwtConfig.TokenDuration = 24 * time.Hour

    // Create a Telegram user (this would come from Telegram WebApp)
    telegramUser := &auth.TelegramUser{
        ID:        123456789,
        FirstName: "John",
        LastName:  "Doe",
        Username:  "johndoe",
        PhotoURL:  "https://example.com/photo.jpg",
        IsBot:     false,
    }

    // Issue JWT token from Telegram user data
    token, err := auth.IssueJWTFromTelegramUser(telegramUser, jwtConfig)
    if err != nil {
        panic(err)
    }

    fmt.Printf("JWT Token: %s\n", token)
}
```

### 2. JWT Middleware for HTTP Handlers

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/quiby-ai/common/pkg/auth"
)

func main() {
    jwtConfig := auth.DefaultJWTConfig("your-secret-key-here")

    // Protected handler that requires JWT authentication
    protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := auth.GetJWTUserFromContext(r.Context())
        if !ok {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        response := map[string]interface{}{
            "message": "Hello, authenticated user!",
            "user_id": claims.UserID,
            "username": claims.Username,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    // Apply JWT middleware
    protectedWithJWT := auth.JWTAuthMiddleware(jwtConfig)(protectedHandler)

    http.HandleFunc("/protected", protectedWithJWT.ServeHTTP)
    http.ListenAndServe(":8080", nil)
}
```

### 3. Complete Telegram to JWT Flow

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/quiby-ai/common/pkg/auth"
)

func main() {
    botToken := "your-telegram-bot-token"
    jwtConfig := auth.DefaultJWTConfig("your-jwt-secret-key")

    // 1. Telegram authentication endpoint (issues JWT)
    telegramAuthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        telegramUser, ok := auth.GetUserFromContext(r.Context())
        if !ok {
            http.Error(w, "Telegram user not found", http.StatusUnauthorized)
            return
        }

        token, err := auth.IssueJWTFromTelegramUser(telegramUser, jwtConfig)
        if err != nil {
            http.Error(w, "Failed to issue JWT", http.StatusInternalServerError)
            return
        }

        response := map[string]interface{}{
            "token": token,
            "user":  telegramUser,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    // 2. Protected API endpoint (requires JWT)
    apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := auth.GetJWTUserFromContext(r.Context())
        if !ok {
            http.Error(w, "JWT user not found", http.StatusUnauthorized)
            return
        }

        response := map[string]interface{}{
            "message": "Access granted to protected API",
            "user_id": claims.UserID,
            "name":    claims.FirstName + " " + claims.LastName,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    // Apply middleware
    telegramAuthWithJWT := auth.TelegramAuthMiddleware(botToken)(telegramAuthHandler)
    protectedAPI := auth.JWTAuthMiddleware(jwtConfig)(apiHandler)

    // Set up routes
    http.HandleFunc("/auth/telegram", telegramAuthWithJWT.ServeHTTP)
    http.HandleFunc("/api/protected", protectedAPI.ServeHTTP)

    http.ListenAndServe(":8080", nil)
}
```

## API Reference

### Types

#### JWTConfig
Configuration for JWT operations.

```go
type JWTConfig struct {
    SecretKey     string        // Secret key for signing JWT tokens
    TokenDuration time.Duration // Token expiration duration
    Issuer        string        // Token issuer
}
```

#### JWTCustomClaims
Custom claims for JWT tokens containing Telegram user data.

```go
type JWTCustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}
```

### Functions

#### DefaultJWTConfig
Creates a default JWT configuration with 24-hour token duration.

```go
func DefaultJWTConfig(secretKey string) *JWTConfig
```

#### IssueJWTFromTelegramUser
Creates a JWT token from Telegram user data.

```go
func IssueJWTFromTelegramUser(user *TelegramUser, config *JWTConfig) (string, error)
```

#### JWTAuthMiddleware
Returns HTTP middleware that requires valid JWT authentication.

```go
func JWTAuthMiddleware(config *JWTConfig) func(http.Handler) http.Handler
```

#### JWTOptionalMiddleware
Returns HTTP middleware that optionally validates JWT tokens.

```go
func JWTOptionalMiddleware(config *JWTConfig) func(http.Handler) http.Handler
```

#### GetJWTUserFromContext
Retrieves JWT user claims from HTTP request context.

```go
func GetJWTUserFromContext(ctx context.Context) (*JWTCustomClaims, bool)
```

#### ValidateJWTToken
Validates a JWT token and returns the claims.

```go
func ValidateJWTToken(tokenString string, config *JWTConfig) (*JWTCustomClaims, error)
```

#### RefreshJWTToken
Creates a new JWT token with extended expiration.

```go
func RefreshJWTToken(tokenString string, config *JWTConfig) (string, error)
```

## Usage Patterns

### 1. Required Authentication
Use `JWTAuthMiddleware` for endpoints that require authentication:

```go
protectedHandler := auth.JWTAuthMiddleware(jwtConfig)(yourHandler)
```

### 2. Optional Authentication
Use `JWTOptionalMiddleware` for endpoints that work with or without authentication:

```go
optionalHandler := auth.JWTOptionalMiddleware(jwtConfig)(yourHandler)
```

### 3. Token Refresh
Implement a refresh endpoint to extend token validity:

```go
refreshHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    
    newToken, err := auth.RefreshJWTToken(tokenString, jwtConfig)
    if err != nil {
        http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
        return
    }
    
    response := map[string]interface{}{"token": newToken}
    json.NewEncoder(w).Encode(response)
})
```

## Security Considerations

1. **Secret Key**: Use a strong, randomly generated secret key for JWT signing
2. **Token Duration**: Set appropriate token expiration times based on your security requirements
3. **HTTPS**: Always use HTTPS in production to protect JWT tokens in transit
4. **Token Storage**: Store JWT tokens securely on the client side (e.g., in memory or secure storage)
5. **Token Validation**: Always validate tokens on the server side before processing requests

## Testing

Run the test suite to verify functionality:

```bash
go test ./pkg/auth -v
```