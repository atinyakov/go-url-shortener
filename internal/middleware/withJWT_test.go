package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInjectUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	userID := "abc123"
	newReq := InjectUserID(req, userID)

	val := newReq.Context().Value(UserIDKey)
	require.Equal(t, userID, val)
}

func TestWithJWT(t *testing.T) {
	t.Run("no token cookie – generate new token", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuth := mocks.NewMockAuthIface(ctrl)

		wantUserID := "generated-user-id"
		wantToken := "mock-token"

		mockAuth.EXPECT().
			BuildJWTString().
			Return(wantToken, wantUserID, nil)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		var gotUserID string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUserID = r.Context().Value(UserIDKey).(string)
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithJWT(mockAuth)(handler)
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
		assert.Equal(t, wantUserID, gotUserID)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "token", cookies[0].Name)
		assert.Equal(t, wantToken, cookies[0].Value)
	})

	t.Run("valid token cookie – parse claims", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuth := mocks.NewMockAuthIface(ctrl)
		userID := "existing-user-id"
		token := "valid-token"

		mockCookie := &http.Cookie{Name: "token", Value: token}

		mockAuth.EXPECT().
			ParseClaims(mockCookie).
			Return(&service.Claims{UserID: userID}, nil)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(mockCookie)
		rec := httptest.NewRecorder()

		var gotUserID string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUserID = r.Context().Value(UserIDKey).(string)
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithJWT(mockAuth)(handler)
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
		assert.Equal(t, userID, gotUserID)
	})

	t.Run("token generation error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuth := mocks.NewMockAuthIface(ctrl)

		mockAuth.EXPECT().
			BuildJWTString().
			Return("", "", errors.New("fail"))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called on error")
		})

		middleware := WithJWT(mockAuth)(handler)
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
	})

	t.Run("token parse error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuth := mocks.NewMockAuthIface(ctrl)
		token := "bad-token"
		mockCookie := &http.Cookie{Name: "token", Value: token}

		mockAuth.EXPECT().
			ParseClaims(mockCookie).Return(nil, errors.New("invalid token"))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(mockCookie)
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called on error")
		})

		middleware := WithJWT(mockAuth)(handler)
		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
	})
}
