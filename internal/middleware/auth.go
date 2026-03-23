package middleware

import (
	"errors"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/response"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	errMissingToken = errors.New("missing token")
	errInvalidToken = errors.New("invalid token")
)

// Claims 表示 JWT 声明
type Claims struct {
	Username string `json:"username"`
	IsOwner  bool   `json:"is_owner"`
	jwt.RegisteredClaims
}

// GenerateJWT 为指定用户名生成 JWT 令牌
func GenerateJWT(username string, isOwner bool) (string, error) {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	claims := &Claims{
		Username: strings.TrimSpace(username),
		IsOwner:  isOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(midnight),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret()))
}

// ParseToken validates a JWT string and returns the claims.
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(strings.TrimSpace(tokenStr), claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret()), nil
	})
	if err != nil || !token.Valid {
		return nil, errInvalidToken
	}

	return claims, nil
}

// ParseRequestToken validates the Authorization header on a request.
func ParseRequestToken(r *http.Request) (*Claims, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return nil, errMissingToken
	}

	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return nil, errInvalidToken
	}

	tokenStr := strings.TrimSpace(authHeader[len("Bearer "):])
	if tokenStr == "" {
		return nil, errInvalidToken
	}

	return ParseToken(tokenStr)
}

// Auth 是验证 JWT 令牌的中间件
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := ParseRequestToken(r); err != nil {
			if errors.Is(err, errMissingToken) {
				response.Unauthorized(w, "Missing token")
				return
			}

			response.Unauthorized(w, "Invalid token")
			return
		}

		next.ServeHTTP(w, r)
	}
}

// OwnerOnly ensures the requester has a valid owner token.
func OwnerOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := ParseRequestToken(r)
		if err != nil {
			if errors.Is(err, errMissingToken) {
				response.Unauthorized(w, "Missing token")
				return
			}

			response.Unauthorized(w, "Invalid token")
			return
		}

		if !claims.IsOwner {
			response.Forbidden(w, "Owner access required")
			return
		}

		next.ServeHTTP(w, r)
	}
}
