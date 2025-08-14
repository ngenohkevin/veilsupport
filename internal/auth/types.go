package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ngenohkevin/veilsupport/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *db.DB
	jwtSecret string
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewAuthService(database *db.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        database,
		jwtSecret: jwtSecret,
	}
}

func (a *AuthService) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}
	
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	
	return string(bytes), nil
}

func (a *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (a *AuthService) GenerateToken(userID int, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, nil
}

func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token cannot be empty")
	}
	
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("failed to parse claims")
	}
	
	return claims, nil
}

func (a *AuthService) Register(email, password string) (*db.User, string, error) {
	// Check if user already exists
	existing, err := a.db.GetUserByEmail(email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, "", errors.New("email already registered")
	}
	
	// Hash password
	hash, err := a.HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	
	// Create user
	user, err := a.db.CreateUser(email, hash)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}
	
	// Generate token
	token, err := a.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	
	return user, token, nil
}

func (a *AuthService) Login(email, password string) (*db.User, string, error) {
	// Get user by email
	user, err := a.db.GetUserByEmail(email)
	if err != nil {
		return nil, "", fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, "", errors.New("invalid credentials")
	}
	
	// Check password
	if !a.CheckPassword(password, user.PasswordHash) {
		return nil, "", errors.New("invalid credentials")
	}
	
	// Generate token
	token, err := a.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	
	return user, token, nil
}