package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/models"
)

func TestBuildJWTString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLService := mocks.NewMockURLServiceIface(ctrl)

	// Mock GetURLByUserID to return empty list to simulate unique user ID
	mockURLService.EXPECT().
		GetURLByUserID(gomock.Any(), gomock.Any()).
		Return(&[]models.ByIDRequest{}, nil)

	auth := service.NewAuth(mockURLService)

	tokenStr, userID, err := auth.BuildJWTString()

	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)
	require.NotEmpty(t, userID)

	// Decode token to verify claims
	token, err := jwt.ParseWithClaims(tokenStr, &service.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("supersecretkey"), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*service.Claims)
	require.True(t, ok)
	require.Equal(t, userID, claims.UserID)
	require.WithinDuration(t, time.Now().Add(service.TokenExp), claims.ExpiresAt.Time, time.Minute)
}

func TestParseClaims(t *testing.T) {
	auth := service.NewAuth(nil) // no need for URLServiceIface

	t.Run("valid token", func(t *testing.T) {
		// Create a test token
		userID := "test-user-id"
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, service.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(service.TokenExp)),
			},
			UserID: userID,
		})

		signedToken, err := token.SignedString([]byte("supersecretkey"))
		require.NoError(t, err)

		cookie := &http.Cookie{
			Name:  "token",
			Value: signedToken,
		}

		claims, err := auth.ParseClaims(cookie)
		require.NoError(t, err)
		require.Equal(t, userID, claims.UserID)
	})

	t.Run("invalid token", func(t *testing.T) {
		cookie := &http.Cookie{
			Name:  "token",
			Value: "invalid.token.here",
		}

		claims, err := auth.ParseClaims(cookie)
		require.Error(t, err)
		require.Nil(t, claims)
	})
}
