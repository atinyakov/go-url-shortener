// Package service provides functionality for handling authentication,
// including generating and parsing JWT tokens. It is designed to interact
// with the URL service to manage user identification and session management.
package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// AuthIface defines the interface for JWT authentication used in middleware.
type AuthIface interface {
	BuildJWTString() (string, string, error)
	ParseClaims(c *http.Cookie) (*Claims, error)
	ParseRawJWT(tokenString string) (*Claims, error)
}

// Claims represents the claims that are included in the JWT token.
// It embeds the RegisteredClaims from the JWT package and includes
// a custom UserID field.
type Claims struct {
	// Embedded RegisteredClaims provides standard JWT claims like Expiration, Issuer, etc.
	jwt.RegisteredClaims
	// UserID is a custom claim for storing the user ID.
	UserID string `json:"user_id"`
}

// TokenExp defines the expiration time of the JWT token (1 year).
const TokenExp = time.Hour * 24 * 365 // 1 year

// secretKey is used for signing JWT tokens. It should be kept private.
const secretKey = "supersecretkey"

// Auth provides methods for building and parsing JWT tokens,
// as well as handling user authentication.
type Auth struct {
	// s is the URL service interface, used for interacting with the storage backend.
	s URLServiceIface
}

// NewAuth creates a new Auth instance, initializing it with the given URLServiceIface.
func NewAuth(s URLServiceIface) *Auth {
	return &Auth{
		s: s,
	}
}

// BuildJWTString generates a new JWT token for the user. It creates a unique user ID
// and returns a JWT token string along with the user ID.
func (a Auth) BuildJWTString() (string, string, error) {
	// Set a timeout context for interacting with the service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var userID string // Variable for storing the generated user ID

	// Generate a unique user ID and ensure it does not already exist in the system
	for {
		tempID := uuid.New().String() // Generate a temporary UUID
		if res, _ := a.s.GetURLByUserID(ctx, tempID); len(*res) == 0 {
			userID = tempID // Use the ID if it's unique
			break
		}
	}

	// Create a new JWT token with the generated user ID and set the expiration date
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)), // Set expiration
		},
		UserID: userID, // Set the custom UserID claim
	})

	// Sign the token and return the string representation of the token
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	return tokenString, userID, nil // Return the JWT token and the user ID
}

// ParseClaims parses the JWT token from the provided HTTP cookie and returns
// the claims embedded within the token.
func (a Auth) ParseClaims(c *http.Cookie) (*Claims, error) {
	// Parse the JWT token from the cookie value
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(c.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil // Provide the signing key for verification
	})

	// If there's an error or the token is invalid, return an error
	if err != nil || !token.Valid {
		return nil, err
	}

	// Return the parsed claims
	return claims, nil
}

func (a *Auth) ParseRawJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token or claims")
	}

	return claims, nil
}
