// Package middleware provides HTTP middleware functions used for
// injecting the user ID into the request context and handling JWT authentication.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
)

// ContextKey is a custom type used for keys in the context.
// It helps prevent collisions in context keys.
type ContextKey string

// UserIDKey is the key used to store and retrieve the user ID from the context.
const UserIDKey ContextKey = "userID"

// InjectUserID adds the user ID to the request context, making it accessible for
// downstream handlers.
func InjectUserID(req *http.Request, userID string) *http.Request {
	// Create a new context with the user ID and attach it to the request.
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	return req.WithContext(ctx)
}

// WithJWT is an HTTP middleware that checks for a valid JWT in the request's cookies.
// If the JWT is missing or invalid, a new one is generated and sent to the client.
// It also injects the user ID from the JWT claims into the request context.
func WithJWT(auth service.AuthIface) func(next http.Handler) http.Handler {
	// Returns a handler that processes the JWT and sets the user ID in the context.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Try to get the "token" cookie from the request.
			cookie, cErr := r.Cookie("token")
			userID := ""

			// If there's no token cookie, generate a new JWT token and set it in the response.
			if cErr != nil {
				tokenString, generatedID, err := auth.BuildJWTString()
				if err != nil {
					// If an error occurs while generating the JWT, return a server error.
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// Set the token cookie with an expiration time.
				http.SetCookie(w, &http.Cookie{
					Name:     "token",
					Value:    tokenString,
					Expires:  time.Now().Add(service.TokenExp),
					HttpOnly: true,
					Path:     "/",
				})

				// Set the user ID from the generated JWT.
				userID = generatedID
			}

			// If a token cookie is present, parse its claims.
			if cookie != nil {
				claims, err := auth.ParseClaims(cookie)
				if err != nil {
					// If there is an error parsing the JWT, return a server error.
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// Set the user ID from the claims.
				userID = claims.UserID
			}

			// Inject the user ID into the request context.
			ctx := context.WithValue(r.Context(), UserIDKey, userID)

			// Call the next handler with the updated request context.
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
