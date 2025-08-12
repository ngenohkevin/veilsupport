# VeilSupport - Anonymous Support Chat System
*Standalone XMPP-based support chat for darkweb integration*

## Project Overview

VeilSupport is a standalone Go service that bridges web interfaces with XMPP for secure, anonymous support chat. Users chat via web interfaces on multiple sites, while admins use standard XMPP clients on any device.

### Core Features
- REST API for easy site integration
- Persistent chat history via email/password
- Real-time WebSocket updates
- XMPP backend with standard clients
- Tor-ready architecture
- Multi-site support from single service

---

## Phase 1: Project Setup & Foundation
**Duration: 1-2 days**

### 1.1 Project Structure
```
veilsupport/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   ├── auth/
│   ├── chat/
│   ├── config/
│   ├── database/
│   └── xmpp/
├── web/
│   └── static/
├── configs/
├── scripts/
└── docs/
```

### 1.2 Dependencies (Production-Ready Stack)
```go
// go.mod
module veilsupport

go 1.21

require (
    // Web framework & WebSocket
    github.com/gin-gonic/gin v1.10.0
    github.com/gorilla/websocket v1.5.1
    
    // XMPP
    mellium.im/xmpp v0.21.4
    
    // Database stack (PostgreSQL + sqlc + migrations)
    github.com/jackc/pgx/v5 v5.5.1
    github.com/jackc/pgx/v5/pgxpool v1.2.1
    github.com/golang-migrate/migrate/v4 v4.17.0
    github.com/golang-migrate/migrate/v4/database/postgres v4.17.0
    github.com/golang-migrate/migrate/v4/source/file v4.17.0
    
    // Authentication & Security
    github.com/golang-jwt/jwt/v5 v5.2.0
    golang.org/x/crypto v0.18.0
    github.com/google/uuid v1.5.0
    
    // Configuration & Environment
    github.com/joho/godotenv v1.5.1
    github.com/spf13/viper v1.18.2
    
    // Logging & Monitoring
    github.com/sirupsen/logrus v1.9.3
    go.uber.org/zap v1.26.0
    
    // Rate limiting & Middleware
    github.com/ulule/limiter/v3 v3.11.2
    github.com/gin-contrib/cors v1.5.0
    
    // Testing
    github.com/stretchr/testify v1.8.4
    github.com/DATA-DOG/go-sqlmock v1.5.2
    github.com/testcontainers/testcontainers-go v0.26.0
    
    // Validation
    github.com/go-playground/validator/v10 v10.16.0
)
```

### 1.3 Testing Tools
```go
// Additional test dependencies
require (
    github.com/stretchr/testify v1.8.4
    github.com/DATA-DOG/go-sqlmock v1.5.2
    github.com/testcontainers/testcontainers-go v0.26.0
    github.com/testcontainers/testcontainers-go/modules/postgres v0.26.0
    github.com/gavv/httpexpect/v2 v2.16.0
)
```

### 1.4 Database Schema Setup (sqlc + migrations)
```bash
# Install sqlc for type-safe SQL
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Directory structure for database
mkdir -p db/{migrations,queries,sqlc}
```

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries"
    schema: "db/migrations"
    gen:
      go:
        package: "db"
        out: "internal/db"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
```

### 1.5 Configuration Management
```go
// internal/config/config.go
type Config struct {
    Server struct {
        Port     string `mapstructure:"port"`
        Host     string `mapstructure:"host"`
        TorProxy string `mapstructure:"tor_proxy"`
    } `mapstructure:"server"`
    
    XMPP struct {
        Server   string `mapstructure:"server"`
        Admin    string `mapstructure:"admin"`
        Password string `mapstructure:"password"`
        Domain   string `mapstructure:"domain"`
    } `mapstructure:"xmpp"`
    
    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
        DBName   string `mapstructure:"db_name"`
        SSLMode  string `mapstructure:"ssl_mode"`
    } `mapstructure:"database"`
    
    JWT struct {
        Secret string `mapstructure:"secret"`
        TTL    int    `mapstructure:"ttl"`
    } `mapstructure:"jwt"`
    
    Redis struct {
        URL string `mapstructure:"url"`
    } `mapstructure:"redis"`
}
```

### 1.6 Test-Driven Development (TDD) Approach

**Philosophy**: Write tests first, implement functionality second. Each feature follows Red-Green-Refactor cycle.

```bash
# Testing directory structure
tests/
├── unit/                    # Fast isolated tests
│   ├── auth/
│   ├── database/
│   ├── xmpp/
│   └── handlers/
├── integration/             # End-to-end API tests
│   ├── api_test.go
│   ├── websocket_test.go
│   └── xmpp_integration_test.go
├── fixtures/                # Test data
│   ├── users.sql
│   └── messages.sql
├── testhelpers/             # Shared test utilities
│   ├── database.go
│   ├── xmpp_mock.go
│   └── server.go
└── load/                    # Performance tests
    └── concurrent_users_test.go
```

#### TDD Workflow for Each Feature
1. **Red**: Write failing test that defines expected behavior
2. **Green**: Write minimal code to make test pass
3. **Refactor**: Clean up code while keeping tests green
4. **Repeat**: Add more test cases and functionality

#### Test Categories
- **Unit Tests**: Test individual functions/methods in isolation
- **Integration Tests**: Test component interactions with real database
- **End-to-End Tests**: Test complete user workflows via API
- **Load Tests**: Verify performance under concurrent load

```go
// Example TDD cycle for user authentication
func TestUserRegistration_Success(t *testing.T) {
    // Setup
    db := testhelpers.SetupTestDB(t)
    defer testhelpers.CleanupTestDB(t, db)
    
    authService := auth.NewService(db)
    
    // Test data
    email := "test@example.com"
    password := "securepassword123"
    
    // Execute
    user, err := authService.Register(email, password)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, email, user.Email)
    assert.NotEmpty(t, user.ID)
    assert.NotEqual(t, password, user.Password) // Should be hashed
}
```

---

## Phase 2: Database Schema & Migrations (TDD First)
**Duration: 1-2 days**

### 2.1 PostgreSQL Migrations
```sql
-- db/migrations/000001_initial_schema.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    xmpp_jid VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    from_jid VARCHAR(255) NOT NULL,
    to_jid VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    message_type VARCHAR(50) NOT NULL, -- 'user' or 'admin'
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_sent_at ON messages(sent_at);
```

### 2.2 SQLC Queries (Type-Safe SQL)
```sql
-- db/queries/users.sql
-- name: CreateUser :one
INSERT INTO users (email, password_hash, xmpp_jid)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2;
```

```sql
-- db/queries/chat_sessions.sql
-- name: CreateChatSession :one
INSERT INTO chat_sessions (user_id)
VALUES ($1)
RETURNING *;

-- name: GetActiveSessionByUserID :one
SELECT * FROM chat_sessions 
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateSessionStatus :exec
UPDATE chat_sessions SET status = $1, updated_at = NOW() WHERE id = $2;
```

```sql
-- db/queries/messages.sql
-- name: SaveMessage :one
INSERT INTO messages (session_id, from_jid, to_jid, content, message_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMessagesBySession :many
SELECT * FROM messages 
WHERE session_id = $1 
ORDER BY sent_at ASC;

-- name: GetRecentMessagesByUserID :many
SELECT m.* FROM messages m
JOIN chat_sessions cs ON m.session_id = cs.id
WHERE cs.user_id = $1
ORDER BY m.sent_at DESC
LIMIT $2;
```

### 2.3 Generated Database Code
```bash
# Generate type-safe Go code from SQL
sqlc generate
```

### 2.4 Database Service Layer (TDD Implementation)
```go
// internal/database/service.go
type Service struct {
    db      *pgxpool.Pool
    queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
    return &Service{
        db:      pool,
        queries: db.New(pool),
    }
}

// Test-driven implementation
func (s *Service) CreateUser(ctx context.Context, email, passwordHash, jid string) (*db.User, error) {
    return s.queries.CreateUser(ctx, db.CreateUserParams{
        Email:        email,
        PasswordHash: passwordHash,
        XmppJid:      jid,
    })
}
```

---

## Phase 3: XMPP Core Implementation (TDD)
**Duration: 2-3 days**

### 2.1 XMPP Client Manager
```go
// internal/xmpp/client.go
type XMPPManager struct {
    adminClient *xmpp.Client
    userClients map[string]*xmpp.Client
    config      *config.Config
    mu          sync.RWMutex
}

func (x *XMPPManager) ConnectAdmin() error
func (x *XMPPManager) CreateUserSession(email string) (*xmpp.Client, error)
func (x *XMPPManager) SendMessage(from, to, message string) error
func (x *XMPPManager) HandleIncoming() chan xmpp.Message
```

### 2.2 Message Handler
```go
// internal/xmpp/handler.go
type MessageHandler struct {
    xmppManager *XMPPManager
    chatService *chat.Service
}

func (h *MessageHandler) ProcessUserMessage(userEmail, message string) error
func (h *MessageHandler) ProcessAdminMessage(targetUser, message string) error
func (h *MessageHandler) StartListening()
```

### 2.3 XMPP Server Setup
```yaml
# prosody.cfg.lua
VirtualHost "yoursite.onion"
    authentication = "internal_hashed"
    modules_enabled = {
        "roster", "saslauth", "tls", "dialback",
        "disco", "carbons", "pep", "private",
        "blocklist", "vcard4", "version", "uptime",
        "time", "ping", "register", "mam", "csi_simple"
    }
    
ssl = {
    certificate = "certs/yoursite.onion.crt";
    key = "certs/yoursite.onion.key";
}
```

---

---

## Phase 4: Integration Testing Framework
**Duration: 1-2 days**

### 4.1 Test Database Setup with Testcontainers
```go
// tests/testhelpers/database.go
func SetupTestDB(t *testing.T) *pgxpool.Pool {
    ctx := context.Background()
    
    // Start PostgreSQL container
    postgres, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(5*time.Second)),
    )
    require.NoError(t, err)
    
    // Get connection string
    connStr, err := postgres.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    // Connect to database
    pool, err := pgxpool.New(ctx, connStr)
    require.NoError(t, err)
    
    // Run migrations
    runMigrations(t, connStr)
    
    // Cleanup function
    t.Cleanup(func() {
        pool.Close()
        postgres.Terminate(ctx)
    })
    
    return pool
}

func runMigrations(t *testing.T, connStr string) {
    m, err := migrate.New(
        "file://../../db/migrations",
        connStr,
    )
    require.NoError(t, err)
    
    err = m.Up()
    require.NoError(t, err)
}
```

### 4.2 Integration Test Examples
```go
// tests/integration/user_registration_test.go
func TestUserRegistrationIntegration(t *testing.T) {
    // Setup test server with real database
    pool := testhelpers.SetupTestDB(t)
    server := testhelpers.SetupTestServer(t, pool)
    defer server.Close()
    
    tests := []struct {
        name           string
        email          string
        password       string
        expectedStatus int
        expectError    bool
    }{
        {
            name:           "successful registration",
            email:          "test@example.com",
            password:       "securepassword123",
            expectedStatus: http.StatusCreated,
            expectError:    false,
        },
        {
            name:           "duplicate email",
            email:          "test@example.com",
            password:       "anotherpassword",
            expectedStatus: http.StatusConflict,
            expectError:    true,
        },
        {
            name:           "weak password",
            email:          "test2@example.com",
            password:       "123",
            expectedStatus: http.StatusBadRequest,
            expectError:    true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            payload := map[string]string{
                "email":    tt.email,
                "password": tt.password,
            }
            
            resp := testhelpers.PostJSON(t, server.URL+"/api/v1/auth/register", payload)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
            
            if !tt.expectError {
                var response struct {
                    User struct {
                        ID    string `json:"id"`
                        Email string `json:"email"`
                    } `json:"user"`
                    Token string `json:"token"`
                }
                
                err := json.NewDecoder(resp.Body).Decode(&response)
                assert.NoError(t, err)
                assert.Equal(t, tt.email, response.User.Email)
                assert.NotEmpty(t, response.Token)
            }
        })
    }
}
```

### 4.3 WebSocket Integration Tests
```go
// tests/integration/websocket_test.go
func TestWebSocketRealTimeMessaging(t *testing.T) {
    pool := testhelpers.SetupTestDB(t)
    server := testhelpers.SetupTestServer(t, pool)
    defer server.Close()
    
    // Create test user and get JWT token
    user, token := testhelpers.CreateTestUser(t, server)
    
    // Connect to WebSocket
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token
    ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer ws.Close()
    
    // Setup message reception channel
    messages := make(chan map[string]interface{})
    go func() {
        defer close(messages)
        for {
            var msg map[string]interface{}
            err := ws.ReadJSON(&msg)
            if err != nil {
                return
            }
            messages <- msg
        }
    }()
    
    // Send message via REST API
    payload := map[string]string{
        "message": "Hello from integration test",
    }
    
    resp := testhelpers.PostJSONWithAuth(t, 
        server.URL+"/api/v1/chat/send", 
        payload, 
        token)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Verify WebSocket receives the message
    select {
    case msg := <-messages:
        assert.Equal(t, "message", msg["type"])
        assert.Equal(t, "Hello from integration test", msg["content"])
    case <-time.After(2 * time.Second):
        t.Fatal("Expected WebSocket message not received within timeout")
    }
}
```

### 4.4 XMPP Integration Tests
```go
// tests/integration/xmpp_integration_test.go
func TestXMPPMessageFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping XMPP integration test in short mode")
    }
    
    pool := testhelpers.SetupTestDB(t)
    
    // Setup XMPP test server (using ejabberd container)
    xmppServer := testhelpers.SetupTestXMPPServer(t)
    defer xmppServer.Close()
    
    server := testhelpers.SetupTestServerWithXMPP(t, pool, xmppServer.ConnectionString())
    defer server.Close()
    
    // Test user sends message
    user, token := testhelpers.CreateTestUser(t, server)
    
    payload := map[string]string{
        "message": "Help needed with order #123",
    }
    
    resp := testhelpers.PostJSONWithAuth(t, 
        server.URL+"/api/v1/chat/send", 
        payload, 
        token)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Connect as admin to XMPP and verify message received
    adminClient := testhelpers.ConnectAsXMPPAdmin(t, xmppServer)
    defer adminClient.Close()
    
    // Wait for message
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    receivedMsg := testhelpers.WaitForXMPPMessage(t, ctx, adminClient)
    assert.Contains(t, receivedMsg.Body, "Help needed with order #123")
    assert.Equal(t, user.XmppJid, receivedMsg.From.Bare().String())
}
```

### 4.5 Load Testing
```go
// tests/load/concurrent_users_test.go
func TestConcurrentUserLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    pool := testhelpers.SetupTestDB(t)
    server := testhelpers.SetupTestServer(t, pool)
    defer server.Close()
    
    const numUsers = 100
    const messagesPerUser = 10
    
    var wg sync.WaitGroup
    errors := make(chan error, numUsers)
    
    for i := 0; i < numUsers; i++ {
        wg.Add(1)
        go func(userNum int) {
            defer wg.Done()
            
            // Register user
            email := fmt.Sprintf("user%d@test.com", userNum)
            user, token := testhelpers.CreateTestUserWithCredentials(t, server, email, "password123")
            
            // Connect WebSocket
            ws := testhelpers.ConnectWebSocket(t, server, token)
            defer ws.Close()
            
            // Send multiple messages
            for j := 0; j < messagesPerUser; j++ {
                message := fmt.Sprintf("Message %d from user %d", j+1, userNum)
                err := testhelpers.SendMessage(t, server, token, message)
                if err != nil {
                    errors <- fmt.Errorf("user %d message %d failed: %w", userNum, j+1, err)
                    return
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    var errorCount int
    for err := range errors {
        t.Errorf("Concurrent user error: %v", err)
        errorCount++
    }
    
    if errorCount > 0 {
        t.Fatalf("Load test failed with %d errors out of %d operations", 
            errorCount, numUsers*messagesPerUser)
    }
    
    t.Logf("Successfully handled %d concurrent users sending %d messages each", 
        numUsers, messagesPerUser)
}
```

---

## Popular & Maintained Services Integration

### Production Services Stack
**All services chosen for production-ready applications with strong maintenance records**

#### Database & ORM
- **PostgreSQL 15+**: Most reliable RDBMS, excellent JSON support
- **pgx/v5**: High-performance PostgreSQL driver (2M+ downloads/month)
- **sqlc**: Type-safe SQL code generation (25k+ GitHub stars)
- **golang-migrate**: Database migration tool (14k+ GitHub stars)

#### Web Framework & Middleware  
- **Gin**: Fast HTTP web framework (75k+ GitHub stars, actively maintained)
- **gin-contrib/cors**: CORS middleware for Gin
- **ulule/limiter/v3**: Rate limiting middleware (2k+ stars)

#### Authentication & Security
- **golang-jwt/jwt/v5**: JWT token handling (6k+ stars, official JWT library)
- **golang.org/x/crypto**: Official Go crypto package for bcrypt
- **google/uuid**: UUID generation (5k+ stars)

#### Configuration & Environment
- **spf13/viper**: Configuration management (25k+ stars)
- **joho/godotenv**: Environment variable loading (7k+ stars)

#### Logging & Monitoring
- **sirupsen/logrus**: Structured logging (24k+ stars)
- **uber-go/zap**: High-performance logging alternative (21k+ stars)

#### Testing Infrastructure
- **testify**: Testing toolkit (22k+ stars, Go standard)
- **testcontainers-go**: Integration testing with Docker containers (3k+ stars)
- **go-sqlmock**: SQL driver mock for testing (6k+ stars)

#### XMPP & Real-time
- **mellium.im/xmpp**: Pure Go XMPP library (active development)
- **gorilla/websocket**: WebSocket implementation (22k+ stars)

#### Deployment & DevOps
- **Docker**: Containerization standard
- **PostgreSQL Official Docker Image**: Production-ready database container
- **Traefik** or **Nginx**: Reverse proxy for production

---

## Phase 5: REST API Development (TDD)
**Duration: 2-3 days**

### 4.1 API Routes
```go
// internal/api/routes.go
func SetupRoutes(r *gin.Engine, deps *Dependencies) {
    api := r.Group("/api/v1")
    
    // Authentication
    api.POST("/auth/register", deps.AuthHandler.Register)
    api.POST("/auth/login", deps.AuthHandler.Login)
    
    // Chat operations
    protected := api.Group("/chat")
    protected.Use(deps.AuthMiddleware.ValidateToken())
    protected.POST("/send", deps.ChatHandler.SendMessage)
    protected.GET("/history", deps.ChatHandler.GetHistory)
    protected.GET("/status", deps.ChatHandler.GetStatus)
    
    // WebSocket endpoint
    api.GET("/ws", deps.ChatHandler.WebSocketHandler)
}
```

### 4.2 API Handlers
```go
// internal/api/chat.go
type ChatHandler struct {
    xmppManager *xmpp.XMPPManager
    dbService   *database.Service
    wsClients   map[string]*websocket.Conn
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
    var req struct {
        Message string `json:"message" binding:"required"`
    }
    // Validate, get user from JWT
    // Send via XMPP to admin
    // Save to database
    // Broadcast to WebSocket if connected
}

func (h *ChatHandler) WebSocketHandler(c *gin.Context) {
    // Upgrade connection
    // Authenticate via query param token
    // Handle real-time message delivery
}
```

### 4.3 Integration Endpoints
```go
// For sites to integrate
POST /api/v1/auth/login
Content-Type: application/json
{
    "email": "user@example.com",
    "password": "userpassword"
}

POST /api/v1/chat/send
Authorization: Bearer <jwt_token>
{
    "message": "I need help with my order"
}

GET /api/v1/chat/history
Authorization: Bearer <jwt_token>
// Returns full conversation history
```

---

## Phase 5: WebSocket Real-time Updates
**Duration: 1-2 days**

### 5.1 WebSocket Manager
```go
// internal/api/websocket.go
type WSManager struct {
    clients    map[string]*websocket.Conn // userEmail -> connection
    broadcast  chan WSMessage
    register   chan WSClient
    unregister chan WSClient
    mu         sync.RWMutex
}

type WSMessage struct {
    UserEmail string
    Type      string // message, status
    Content   string
    Timestamp time.Time
}

func (ws *WSManager) Run()
func (ws *WSManager) SendToUser(email, message string)
```

### 5.2 XMPP-WebSocket Bridge
```go
// Bridge admin messages to web users
func (h *ChatHandler) bridgeAdminMessages() {
    for adminMsg := range h.xmppManager.AdminMessages {
        // Parse target user from admin message
        // Send via WebSocket to user if online
        // Always save to database
        h.wsManager.SendToUser(targetEmail, adminMsg.Content)
    }
}
```

---

## Phase 6: Authentication & Security
**Duration: 1 day**

### 6.1 JWT Authentication
```go
// internal/auth/jwt.go
func GenerateToken(userID uint, email string) (string, error)
func ValidateToken(tokenString string) (*Claims, error)
func AuthMiddleware() gin.HandlerFunc
```

### 6.2 Password Security
```go
// internal/auth/password.go
func HashPassword(password string) (string, error)
func CheckPassword(password, hash string) bool
```

### 6.3 Rate Limiting
```go
// internal/api/middleware.go
func RateLimitMiddleware() gin.HandlerFunc {
    // Limit messages per user per minute
    // Prevent spam/abuse
}
```

---

## Phase 7: Frontend Integration Package
**Duration: 1 day**

### 7.1 JavaScript SDK
```javascript
// veilsupport-client.js
class VeilSupportClient {
    constructor(apiUrl) {
        this.apiUrl = apiUrl;
        this.token = null;
        this.ws = null;
    }
    
    async login(email, password) {}
    async sendMessage(message) {}
    async getHistory() {}
    connectWebSocket(onMessage) {}
    disconnect() {}
}
```

### 7.2 Example Integration
```html
<!-- Any site can integrate -->
<div id="support-chat"></div>
<script src="veilsupport-client.js"></script>
<script>
const support = new VeilSupportClient('http://veilsupport.onion/api/v1');

async function startChat() {
    await support.login(email, password);
    const history = await support.getHistory();
    support.connectWebSocket(handleNewMessage);
}
</script>
```

---

## Phase 8: Deployment & Tor Configuration
**Duration: 1 day**

### 8.1 Docker Setup
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o veilsupport cmd/server/main.go

FROM alpine:latest
RUN apk add --no-cache tor
COPY --from=builder /app/veilsupport /usr/local/bin/
COPY configs/ /etc/veilsupport/
EXPOSE 8080 9050
CMD ["veilsupport"]
```

### 8.2 Tor Configuration
```
# torrc
HiddenServiceDir /var/lib/tor/veilsupport/
HiddenServicePort 80 127.0.0.1:8080
HiddenServiceVersion 3
```

### 8.3 XMPP Server Setup
```bash
# Install prosody
apt install prosody

# Configure for .onion domain
# Enable OMEMO encryption
# Set up admin account
```

---

## Phase 9: Testing & Security Hardening
**Duration: 2 days**

### 9.1 Security Features
- Input validation and sanitization
- SQL injection prevention (GORM handles this)
- XSS protection on web endpoints
- Request size limits
- Connection timeouts
- Encrypted message storage option

### 9.2 Testing Strategy
```go
// tests/
├── integration/
│   ├── api_test.go
│   ├── xmpp_test.go
│   └── websocket_test.go
├── unit/
│   ├── auth_test.go
│   ├── database_test.go
│   └── handlers_test.go
└── load/
    └── concurrent_users_test.go
```

### 9.3 Monitoring
```go
// Add logging, metrics
// Health check endpoint
// XMPP connection monitoring
// Database connection pooling
```

---

## Phase 10: Documentation & Integration Guide
**Duration: 1 day**

### 10.1 API Documentation
- OpenAPI/Swagger specs
- Integration examples for common frameworks
- Error handling guide
- WebSocket protocol documentation

### 10.2 Deployment Guide
- Tor hidden service setup
- XMPP server configuration
- SSL/TLS certificate management
- Backup and recovery procedures

---

## Implementation Timeline (TDD Approach)

| Phase | Duration | Dependencies | TDD Focus |
|-------|----------|-------------|-----------|
| 1. Project Setup & TDD Infrastructure | 1-2 days | None | Test helpers, containers, CI setup |
| 2. Database Schema & Migrations (TDD) | 1-2 days | Phase 1 | Database layer tests, sqlc integration |
| 3. XMPP Core (TDD) | 3-4 days | Phase 1,2 | XMPP client tests, message flow tests |
| 4. Integration Testing Framework | 1-2 days | Phase 1,2 | End-to-end test infrastructure |
| 5. REST API (TDD) | 3-4 days | Phase 2,3,4 | API endpoint tests, middleware tests |
| 6. WebSocket Real-time (TDD) | 2-3 days | Phase 5 | WebSocket connection & message tests |
| 7. Authentication & Security (TDD) | 2 days | Phase 5 | Auth flow tests, security tests |
| 8. Frontend SDK | 1-2 days | Phase 5 | Client library with tests |
| 9. Deployment & Production Setup | 1-2 days | Phase 8 | Docker, environment configs |
| 10. Load Testing & Performance | 1-2 days | Phase 9 | Concurrent user tests, benchmarks |
| 11. Documentation & Examples | 1 day | Phase 10 | API docs, integration guides |

**Total: 15-22 days** (increased due to comprehensive TDD approach)

### TDD Benefits in Timeline
- **Longer initial development**: +25% time investment
- **Reduced debugging time**: Catch issues early
- **Faster feature additions**: Solid test foundation
- **Higher confidence**: Comprehensive test coverage
- **Production stability**: Well-tested codebase

### Continuous Integration Pipeline
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_USER: testuser  
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install sqlc
      run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
      
    - name: Generate sqlc code
      run: sqlc generate
      
    - name: Run migrations
      run: migrate -path db/migrations -database "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable" up
      
    - name: Run unit tests
      run: go test ./tests/unit/... -v
      
    - name: Run integration tests  
      run: go test ./tests/integration/... -v
      
    - name: Run load tests
      run: go test ./tests/load/... -v -timeout=10m
```

### Testing Commands Reference
```bash
# Development workflow
make test-unit           # Fast unit tests only
make test-integration    # Integration tests with containers  
make test-load          # Load/performance tests
make test-all           # Full test suite

# TDD workflow
make test-watch         # Watch files and re-run tests
make test-coverage      # Generate coverage report
make test-race          # Run tests with race detector

# Database operations
make migrate-up         # Apply database migrations
make migrate-down       # Rollback migrations
make sqlc-generate      # Generate type-safe SQL code
```

---

## Integration Examples

### Next.js Site Integration
```typescript
// pages/support.tsx
import { useState, useEffect } from 'react';
import VeilSupportClient from '../lib/veilsupport-client';

export default function SupportPage() {
    const [client] = useState(new VeilSupportClient(process.env.VEILSUPPORT_API));
    const [messages, setMessages] = useState([]);
    
    useEffect(() => {
        client.connectWebSocket((msg) => {
            setMessages(prev => [...prev, msg]);
        });
    }, []);
    
    const sendMessage = async (text: string) => {
        await client.sendMessage(text);
    };
    
    return <ChatInterface messages={messages} onSend={sendMessage} />;
}
```

### PHP Site Integration
```php
// support.php
$veilsupport = new VeilSupportClient('http://veilsupport.onion/api/v1');

if ($_POST['message']) {
    $veilsupport->login($_SESSION['email'], $_SESSION['password']);
    $veilsupport->sendMessage($_POST['message']);
}

$history = $veilsupport->getHistory();
```

---

## Security Considerations

### Message Encryption
- OMEMO for XMPP end-to-end encryption
- TLS for all API communications
- Optional database field encryption for stored messages

### Privacy Protection
- No IP logging
- Minimal metadata collection
- Automatic session cleanup
- Optional message expiration

### Tor Integration
- SOCKS5 proxy support for outbound XMPP connections
- Hidden service configuration
- Circuit isolation for different users

---

## Android XMPP Client Setup (Admin)

### Recommended Client: Conversations
1. Install from F-Droid
2. Add account: admin@yoursite.onion
3. Configure Tor proxy (Orbot)
4. Enable OMEMO encryption
5. Users appear as user123@yoursite.onion contacts

### Message Flow
```
User types on website → API → XMPP → Android notification
Admin replies on Android → XMPP → WebSocket → User sees instantly
```

---

## Deployment Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Darkweb Site  │───▶│   VeilSupport    │───▶│ XMPP Server     │
│   (Next.js)     │    │   (Go Service)   │    │ (Prosody)       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                           │
┌─────────────────┐           │                           │
│   Darkweb Site  │───────────┘                           │
│   (PHP)         │                                       │
└─────────────────┘                                       │
                                                          │
┌─────────────────┐                                       │
│   Android       │◄──────────────────────────────────────┘
│   XMPP Client   │
└─────────────────┘
```

---

## Success Metrics

- **Functionality**: Users can chat, admins receive messages instantly
- **Persistence**: Chat history survives browser sessions
- **Integration**: Any site can integrate with <10 lines of code
- **Security**: All communications encrypted, no metadata leaks
- **Performance**: Sub-second message delivery, supports 100+ concurrent users
- **Reliability**: 99.9% uptime, automatic reconnection handling

---

## Next Steps After Implementation

1. **Load Testing**: Simulate high user volumes
2. **Security Audit**: Third-party penetration testing
3. **Feature Extensions**: File sharing, typing indicators
4. **Multi-language Support**: Internationalization
5. **Advanced Analytics**: Support metrics dashboard (privacy-preserving)

---

## File Structure Output
After completion, the project will generate these key files for integration:

- `veilsupport` (Go binary)
- `veilsupport-client.js` (Frontend SDK)
- `integration-examples/` (Next.js, PHP, vanilla JS examples)
- `docker-compose.yml` (Full stack deployment)
- `API.md` (Complete API documentation)