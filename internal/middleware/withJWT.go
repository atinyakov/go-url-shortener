package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
)

type ContextKey string

const UserIDKey ContextKey = "userID"

func InjectUserID(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	return req.WithContext(ctx)
}

func WithJWT(auth *service.Auth) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cookie, cErr := r.Cookie("token")
			userID := ""

			if cErr != nil {

				tokenString, generatedID, err := auth.BuildJWTString()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "token",
					Value:    tokenString,
					Expires:  time.Now().Add(service.TokenExp),
					HttpOnly: true,
					Path:     "/",
				})

				userID = generatedID
			}

			if cookie != nil {
				claims, err := auth.ParseClaims(cookie)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				userID = claims.UserID
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
