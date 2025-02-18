package service

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

const TokenExp = time.Hour * 24 * 365 // 1 year
const secretKey = "supersecretkey"

type Auth struct {
	s *URLService
}

func NewAuth(s *URLService) *Auth {
	return &Auth{
		s: s,
	}
}

func (a Auth) BuildJWTString() (string, string, error) {

	var userID string // Replace with database lookup if user exists

	for {
		tempID := uuid.New().String()
		if res, _ := a.s.GetURLByUserID(tempID); len(*res) == 0 {
			userID = tempID
			break
		}
	}

	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	// возвращаем строку токена
	return tokenString, userID, nil
}

func (a Auth) ParseClaims(c *http.Cookie) (*Claims, error) {
	// Parse token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(c.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
