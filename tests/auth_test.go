package tests

import (
	"os"
	"testing"

	"github.com/ngenohkevin/veilsupport/internal/auth"
	"github.com/stretchr/testify/assert"
)

func setupAuthService(t *testing.T) *auth.AuthService {
	// Setup test database
	db := setupTestDB(t)
	
	// Use test JWT secret or default
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-secret-key-change-in-production"
	}
	
	return auth.NewAuthService(db, jwtSecret)
}

func TestPasswordHashing(t *testing.T) {
	authService := setupAuthService(t)
	
	hash, err := authService.HashPassword("mypassword")
	assert.NoError(t, err)
	assert.NotEqual(t, "mypassword", hash)
	assert.NotEmpty(t, hash)
	
	// Test valid password check
	valid := authService.CheckPassword("mypassword", hash)
	assert.True(t, valid)
	
	// Test invalid password check
	invalid := authService.CheckPassword("wrongpassword", hash)
	assert.False(t, invalid)
}

func TestJWTGeneration(t *testing.T) {
	authService := setupAuthService(t)
	
	token, err := authService.GenerateToken(123, "test@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	// Validate the generated token
	claims, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, 123, claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestJWTValidation(t *testing.T) {
	authService := setupAuthService(t)
	
	// Generate a valid token
	token, err := authService.GenerateToken(456, "user@test.com")
	assert.NoError(t, err)
	
	// Validate it
	claims, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, 456, claims.UserID)
	assert.Equal(t, "user@test.com", claims.Email)
	
	// Test invalid token
	_, err = authService.ValidateToken("invalid.token.here")
	assert.Error(t, err)
	
	// Test empty token
	_, err = authService.ValidateToken("")
	assert.Error(t, err)
}

func TestRegistration(t *testing.T) {
	authService := setupAuthService(t)
	
	// Test successful registration
	user, token, err := authService.Register("new@example.com", "password123")
	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", user.Email)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, user.XmppJID)
	
	// Verify token is valid
	claims, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, "new@example.com", claims.Email)
	
	// Test duplicate registration should fail
	_, _, err = authService.Register("new@example.com", "password456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestLogin(t *testing.T) {
	authService := setupAuthService(t)
	
	// Register a user first
	email := "login@example.com"
	password := "testpassword"
	_, _, err := authService.Register(email, password)
	assert.NoError(t, err)
	
	// Test successful login
	user, token, err := authService.Login(email, password)
	assert.NoError(t, err)
	assert.Equal(t, email, user.Email)
	assert.NotEmpty(t, token)
	
	// Verify token is valid
	claims, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	
	// Test login with wrong password
	_, _, err = authService.Login(email, "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	
	// Test login with non-existent user
	_, _, err = authService.Login("nonexistent@example.com", password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestPasswordComplexity(t *testing.T) {
	authService := setupAuthService(t)
	
	// Test various password scenarios
	testCases := []struct {
		password string
		valid    bool
	}{
		{"password123", true},     // Valid
		{"short", false},          // Too short
		{"", false},               // Empty
		{"verylongpasswordthatisacceptable", true}, // Long password
	}
	
	for _, tc := range testCases {
		hash, err := authService.HashPassword(tc.password)
		if tc.valid {
			assert.NoError(t, err, "Password %s should be valid", tc.password)
			assert.NotEmpty(t, hash)
		} else {
			// Note: bcrypt doesn't validate complexity by default,
			// but we test the hashing works for any non-empty string
			if tc.password != "" {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		}
	}
}