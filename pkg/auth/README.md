# Auth Package

The `auth` package provides ready-to-use primitives for authentication in services.

## Main Features

- ✅ Telegram `initData` validation via middleware
- ✅ Issue and validate short access JWT tokens (HS256)
- ✅ Simple `RequireAuth` middleware for JWT
- ✅ Middleware for Telegram authentication

## Installation

```bash
go get github.com/your-org/common/pkg/auth
```

## Usage

### 1. Telegram Authentication

```go
import "github.com/your-org/common/pkg/auth"

func main() {
    botToken := "your-bot-token"
    
    mux := http.NewServeMux()
    mux.HandleFunc("/protected", protectedHandler)
    
    // Protect route with Telegram middleware
    protectedMux := auth.TelegramAuthMiddleware(botToken)(mux)
    
    http.ListenAndServe(":8080", protectedMux)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := auth.GetUserFromContext(r.Context())
    if !ok {
        http.Error(w, "User not found in context", http.StatusInternalServerError)
        return
    }
    
    w.Write([]byte(fmt.Sprintf("Hello, %s!", user.FirstName)))
}
```

### 2. Issue and Validate JWT Tokens

```go
import "github.com/your-org/common/pkg/auth"

func issueToken(w http.ResponseWriter, r *http.Request) {
    cfg := &auth.JWTConfig{
        Issuer:    "your-service",
        Audience:  "your-app",
        AccessTTL: 1 * time.Hour,
        SecretKey: []byte("your-secret-key"),
    }
    
    user := auth.UserIdentity{UserID: "12345"}
    token, err := auth.IssueAccessJWT(user, cfg)
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    
    // Return token to client
    w.Write([]byte(token))
}

func validateToken(token string) (string, error) {
    cfg := &auth.JWTConfig{
        SecretKey: []byte("your-secret-key"),
    }
    
    userID, err := auth.ValidateAccessJWT(token, cfg)
    if err != nil {
        return "", err
    }
    
    return userID, nil
}
```

### 3. JWT Middleware RequireAuth

```go
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := auth.GetUserIDFromContext(r.Context())
    if !ok {
        http.Error(w, "User not found in context", http.StatusInternalServerError)
        return
    }
    
    w.Write([]byte(fmt.Sprintf("Hello, user %s!", userID)))
}

func main() {
    cfg := &auth.JWTConfig{
        SecretKey: []byte("your-secret-key"),
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/protected", protectedHandler)
    
    // Protect route with JWT middleware
    protectedMux := auth.RequireAuth(cfg, mux)
    
    http.ListenAndServe(":8080", protectedMux)
}
```

## Data Structures

### JWTConfig

```go
type JWTConfig struct {
    Issuer    string            // Token issuer
    Audience  string            // Token audience
    AccessTTL time.Duration     // Token lifetime
    SecretKey []byte            // Secret key for HS256
}
```

### UserIdentity

```go
type UserIdentity struct {
    UserID string // User ID (string)
}
```

### TelegramUser

```go
type TelegramUser struct {
    ID        int64  `json:"id"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name,omitempty"`
    Username  string `json:"username,omitempty"`
    PhotoURL  string `json:"photo_url,omitempty"`
    IsBot     bool   `json:"is_bot,omitempty"`
}
```

## JWT Claims

Access token contains the following claims:

- `sub`: User ID
- `iss`: Token issuer
- `aud`: Token audience
- `iat`: Issued at time
- `exp`: Expiration time
- `jti`: Unique token ID (16 bytes)

## Telegram Authentication

Telegram middleware expects `Authorization: tma <init-data>` header and validates it using:

- HMAC-SHA256 signature with botToken
- Time validation (24 hours)
- Bot check
- User data parsing

## Security

- ✅ HMAC-SHA256 signature for Telegram initData
- ✅ Time validation (24 hours)
- ✅ Bot check
- ✅ JWT with HS256 algorithm
- ✅ Unique token IDs

## Testing

```bash
cd pkg/auth
go test -v
```

## Examples

Complete usage examples can be found in tests:

- `jwt_test.go` - JWT functionality tests
- `telegram_test.go` - Telegram middleware tests

## Principles

The package follows these principles:

- **KISS** - simple and clear API
- **DRY** - reusable components
- **Single Responsibility** - each function does one thing
- **Composition over Inheritance** - using middleware pattern

## Dependencies

- `github.com/golang-jwt/jwt/v5` - for JWT tokens
- `github.com/telegram-mini-apps/init-data-golang` - for Telegram initData validation
